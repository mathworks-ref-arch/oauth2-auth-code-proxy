// Copyright (c) 2020-2026, The MathWorks, Inc. All rights reserved.

package admin

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mathworks-ref-arch/container-images/oauth2-auth-code-proxy/internal/domain"
	"go.uber.org/zap"
)

var getenvFunc = os.Getenv

type URLStatus struct {
	URL      string            `json:"url"`
	ViaProxy bool              `json:"viaProxy"`
	Times    URLStatusTimes    `json:"times"`
	Status   URLStatusMetadata `json:"status"`
}

type URLStatusTimes struct {
	DNSLookup        string `json:"dnsLookup"`
	TCPConnection    string `json:"tcpConnection"`
	Ping             string `json:"ping"`
	TLSHandshake     string `json:"tlsHandshake"`
	HTTPResponseTime string `json:"httpResponseTime"`
}

type URLStatusMetadata struct {
	Success     bool   `json:"success"`
	Error       string `json:"error"`
	LastChecked string `json:"lastChecked"`
}

type ProxyConfig struct {
	HTTPProxy  string `json:"httpProxy,omitempty"`
	HTTPSProxy string `json:"httpsProxy,omitempty"`
	NoProxy    string `json:"noProxy,omitempty"`
}

type AliveResponse struct {
	MWType         string       `json:"mwtype"`
	Echo           string       `json:"echo"`
	Version        string       `json:"version"`
	ArtifactID     string       `json:"artifactId"`
	GroupID        string       `json:"groupId"`
	BuildTimestamp string       `json:"buildTimestamp"`
	Proxy          *ProxyConfig `json:"proxy,omitempty"`
	ConnCheck      []URLStatus  `json:"connCheck"`
}

type HealthChecker struct {
	status          sync.Map
	initOnce        sync.Once
	loaded          atomic.Bool
	checkInProgress atomic.Bool
	apps            map[string]domain.AppConfig
	info            domain.BuildInfo
	logger          *zap.Logger
}

func NewHealthChecker(apps map[string]domain.AppConfig, info domain.BuildInfo, logger *zap.Logger) *HealthChecker {
	return &HealthChecker{
		apps:   apps,
		info:   info,
		logger: logger,
	}
}

func (h *HealthChecker) Loaded() bool {
	return h.loaded.Load()
}

func (h *HealthChecker) InitURLs() {
	h.initURLs()
}

func Register(mux *http.ServeMux, prefix string, hc *HealthChecker) {
	a := &aliveHandler{health: hc}
	mux.HandleFunc("GET "+prefix+"/admin/alive", a.handleAlive)
	mux.HandleFunc("GET "+prefix+"/admin/alive/{echo...}", a.handleAlive)
}

type aliveHandler struct {
	health *HealthChecker
}

// @Summary		Health check
// @Description	Returns service version, build info, and connectivity diagnostics for all
// @Description	configured OAuth2 provider endpoints (auth, token, revoke, testing URLs).
// @Tags			admin
// @Produce		json
// @Param			echo	path		string	false	"Echo string returned in response"
// @Success		200		{object}	AliveResponse
// @Router			/admin/alive/{echo} [get]
func (a *aliveHandler) handleAlive(w http.ResponseWriter, r *http.Request) {
	echo := r.PathValue("echo")

	result := a.health.Check(echo)

	domain.WriteJSON(w, result)
}

func (h *HealthChecker) Check(echo string) AliveResponse {
	h.initURLs()
	go h.checkURLs()

	var currentStatus []URLStatus
	h.status.Range(func(_, value any) bool {
		if s, ok := value.(URLStatus); ok {
			currentStatus = append(currentStatus, s)
		}
		return true
	})

	var proxy *ProxyConfig
	if pc := detectProxyConfig(); pc != (ProxyConfig{}) {
		proxy = &pc
	}

	return AliveResponse{
		MWType:         "oauth/AliveResponse",
		Echo:           echo,
		Version:        h.info.Version,
		ArtifactID:     h.info.ModuleName,
		GroupID:        h.info.GroupID,
		BuildTimestamp: h.info.BuildTime,
		Proxy:          proxy,
		ConnCheck:      currentStatus,
	}
}

func detectProxyConfig() ProxyConfig {
	httpsProxy := getenvFunc("HTTPS_PROXY")
	if httpsProxy == "" {
		httpsProxy = getenvFunc("https_proxy")
	}
	httpProxy := getenvFunc("HTTP_PROXY")
	if httpProxy == "" {
		httpProxy = getenvFunc("http_proxy")
	}
	noProxy := getenvFunc("NO_PROXY")
	if noProxy == "" {
		noProxy = getenvFunc("no_proxy")
	}
	return ProxyConfig{
		HTTPProxy:  httpProxy,
		HTTPSProxy: httpsProxy,
		NoProxy:    noProxy,
	}
}

func (h *HealthChecker) initURLs() {
	h.initOnce.Do(func() {
		for _, app := range h.apps {
			for _, u := range []string{app.TokenURL, app.TestingURL, app.RevokeURL, app.AuthURL} {
				urlObj, err := url.Parse(u)
				if err != nil {
					h.status.Store(u, URLStatus{
						URL:    u,
						Status: URLStatusMetadata{Error: err.Error()},
					})
					continue
				}
				baseURL := urlObj.Scheme + "://" + urlObj.Hostname()
				h.status.Store(baseURL, checkURLConnectivity(baseURL))
			}
		}
		h.loaded.Store(true)
	})
}

func (h *HealthChecker) checkURLs() {
	if !h.checkInProgress.CompareAndSwap(false, true) {
		return
	}
	defer h.checkInProgress.Store(false)

	count := 0
	h.status.Range(func(_, _ any) bool {
		count++
		return true
	})
	if count == 0 {
		return
	}

	var wg sync.WaitGroup
	h.status.Range(func(key, _ any) bool {
		u, ok := key.(string)
		if !ok {
			return true
		}
		if !strings.Contains(u, "//") {
			u = "https://" + u
		}
		wg.Add(1)
		go func(target string) {
			defer wg.Done()
			result := checkURLConnectivity(target)
			h.status.Store(target, result)
		}(u)
		return true
	})
	wg.Wait()
}

func resolvePort(u *url.URL) string {
	if p := u.Port(); p != "" {
		return p
	}
	if u.Scheme == "https" {
		return "443"
	}
	return "80"
}

func hasHTTPProxy(target string) bool {
	req, err := http.NewRequest("GET", target, nil)
	if err != nil {
		return false
	}
	proxyURL, err := http.ProxyFromEnvironment(req)
	if err != nil {
		return false
	}
	return proxyURL != nil
}

func checkURLConnectivity(target string) URLStatus {
	if hasHTTPProxy(target) {
		return checkURLViaProxy(target)
	}
	return checkURLDirect(target)
}

func checkURLViaProxy(target string) URLStatus {
	var status URLStatus
	status.URL = target
	status.ViaProxy = true
	status.Status.LastChecked = time.Now().UTC().Format(time.RFC3339)
	status.Times.DNSLookup = "proxy"
	status.Times.TCPConnection = "proxy"
	status.Times.Ping = "proxy"
	status.Times.TLSHandshake = "proxy"

	transport := &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
	}
	client := &http.Client{Transport: transport, Timeout: 10 * time.Second}
	httpStart := time.Now()
	resp, err := client.Get(target)
	status.Times.HTTPResponseTime = fmt.Sprintf("%.3fs", time.Since(httpStart).Seconds())
	if err != nil {
		status.Status.Error = "HTTP error (via proxy): " + err.Error()
		return status
	}
	_ = resp.Body.Close()

	status.Status.Success = true
	return status
}

func checkURLDirect(target string) URLStatus {
	const dialTimeout = 10 * time.Second

	var status URLStatus
	status.URL = target
	status.Status.LastChecked = time.Now().UTC().Format(time.RFC3339)

	parsed, err := url.Parse(target)
	if err != nil {
		status.Status.Error = err.Error()
		return status
	}
	host := parsed.Hostname()
	port := resolvePort(parsed)

	dnsCtx, dnsCancel := context.WithTimeout(context.Background(), dialTimeout)
	defer dnsCancel()
	resolver := &net.Resolver{}
	dnsStart := time.Now()
	addrs, err := resolver.LookupHost(dnsCtx, host)
	status.Times.DNSLookup = fmt.Sprintf("%.3fs", time.Since(dnsStart).Seconds())
	if err != nil {
		status.Status.Error = "DNS error: " + err.Error()
		return status
	}
	if len(addrs) == 0 {
		status.Status.Error = "DNS error: no addresses found"
		return status
	}

	tcpCtx, tcpCancel := context.WithTimeout(context.Background(), dialTimeout)
	defer tcpCancel()
	dialer := &net.Dialer{}
	tcpStart := time.Now()
	addrToDial := net.JoinHostPort(host, port)
	conn, err := dialer.DialContext(tcpCtx, "tcp", addrToDial)
	status.Times.TCPConnection = fmt.Sprintf("%.3fs", time.Since(tcpStart).Seconds())
	if err != nil {
		status.Status.Error = "TCP error: " + err.Error()
		return status
	}
	status.Times.Ping = status.Times.TCPConnection

	if parsed.Scheme == "https" {
		tlsStart := time.Now()
		tlsConn := tls.Client(conn, &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
		err = tlsConn.Handshake()
		status.Times.TLSHandshake = fmt.Sprintf("%.3fs", time.Since(tlsStart).Seconds())
		if err != nil {
			_ = tlsConn.Close()
			status.Status.Error = "TLS error: " + err.Error()
			return status
		}
		_ = tlsConn.Close()
	} else {
		_ = conn.Close()
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, addr)
		},
		TLSClientConfig: &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12},
	}
	client := &http.Client{Transport: transport, Timeout: dialTimeout}
	httpStart := time.Now()
	resp, err := client.Get(target) //nolint:gosec // target is from admin-configured creds.yaml
	status.Times.HTTPResponseTime = fmt.Sprintf("%.3fs", time.Since(httpStart).Seconds())
	if err != nil {
		status.Status.Error = "HTTP error: " + err.Error()
		return status
	}
	_ = resp.Body.Close()

	status.Status.Success = true
	return status
}
