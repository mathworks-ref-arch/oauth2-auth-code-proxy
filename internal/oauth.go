// Copyright (c) 2020-2026, The MathWorks, Inc. All rights reserved.

package proxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/mathworks-ref-arch/container-images/oauth2-auth-code-proxy/internal/domain"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

type HTTPClient = domain.HTTPClient

type TokenSourcer = domain.TokenSourcer

type OAuthService = domain.OAuthService

type TokenResult = domain.TokenResult

type RefreshResult = domain.RefreshResult

type ExpiryInfo = domain.ExpiryInfo

type UserInfo = domain.UserInfo

type HTTPClientConfig = domain.HTTPClientConfig

type OAuthServiceImpl struct {
	apps      map[string]AppConfig
	configs   map[string]*oauth2.Config
	sourcers  map[string]TokenSourcer
	client    HTTPClient
	exchanger func(ctx context.Context, cfg *oauth2.Config, code, verifier string) (*oauth2.Token, error)
	logger    *zap.Logger
	httpCfg   HTTPClientConfig
}

type realTokenSourcer struct {
	config *oauth2.Config
}

func (r *realTokenSourcer) TokenSource(ctx context.Context, t *oauth2.Token) oauth2.TokenSource {
	return r.config.TokenSource(ctx, t)
}

func (r *realTokenSourcer) Config() *oauth2.Config {
	return r.config
}

func NewOAuthService(apps map[string]AppConfig, httpCfg HTTPClientConfig, logger *zap.Logger) OAuthService {
	configs := make(map[string]*oauth2.Config, len(apps))
	sourcers := make(map[string]TokenSourcer, len(apps))

	for id, app := range apps {
		cfg := &oauth2.Config{
			ClientID:     app.ClientID,
			ClientSecret: app.ClientSecret,
			RedirectURL:  app.RedirectURI,
			Scopes:       app.Scopes,
			Endpoint: oauth2.Endpoint{
				AuthURL:  app.AuthURL,
				TokenURL: app.TokenURL,
			},
		}
		configs[id] = cfg
		sourcers[id] = &realTokenSourcer{config: cfg}
	}

	svc := &OAuthServiceImpl{
		apps:     apps,
		configs:  configs,
		sourcers: sourcers,
		logger:   logger,
		httpCfg:  httpCfg,
	}
	svc.client = svc.buildHTTPClient()
	svc.exchanger = defaultExchanger

	return svc
}

func defaultExchanger(ctx context.Context, cfg *oauth2.Config, code, verifier string) (*oauth2.Token, error) {
	return cfg.Exchange(ctx, code, oauth2.VerifierOption(verifier))
}

func (s *OAuthServiceImpl) Authorize(appID, redirectURL string) (string, string, error) {
	app, appOK := s.apps[appID]
	cfg, cfgOK := s.configs[appID]
	if !appOK || !cfgOK {
		s.logger.Error("app not found", zap.String("appID", appID))
		return "", "", fmt.Errorf("app not found: %s", appID)
	}

	verifier := oauth2.GenerateVerifier()

	opts := []oauth2.AuthCodeOption{
		oauth2.ApprovalForce,
		oauth2.S256ChallengeOption(verifier),
	}

	if app.IncludeRefreshToken {
		opts = append(opts, oauth2.AccessTypeOffline)
	} else {
		opts = append(opts, oauth2.AccessTypeOnline)
	}

	state := appID
	if len(redirectURL) > 0 {
		state = fmt.Sprintf("%s-%s", appID, redirectURL)
	}

	return cfg.AuthCodeURL(state, opts...), verifier, nil
}

func (s *OAuthServiceImpl) Token(ctx context.Context, code, state, verifier, scope string) (*TokenResult, error) {
	parts := strings.SplitN(state, "-", 2)
	appID := parts[0]

	app, appOK := s.apps[appID]
	cfg, cfgOK := s.configs[appID]
	if !appOK || !cfgOK {
		return nil, fmt.Errorf("app not found: %s", appID)
	}

	returnedScopeMap := map[string]bool{}
	scopes := []string{}
	for _, sc := range strings.Split(scope, " ") {
		scopes = append(scopes, sc)
		returnedScopeMap[strings.TrimPrefix(strings.TrimPrefix(sc, "https://"), "http://")] = true
	}

	errorScopes := []string{}
	for _, sc := range app.Scopes {
		if _, found := returnedScopeMap[strings.TrimPrefix(strings.TrimPrefix(sc, "https://"), "http://")]; !found {
			errorScopes = append(errorScopes, sc)
		}
	}
	if len(errorScopes) > 0 {
		sort.Strings(errorScopes)
		s.logger.Error("missing required scopes",
			zap.String("appID", appID),
			zap.String("required", strings.Join(app.Scopes, ", ")),
			zap.String("returned", strings.Join(scopes, ", ")),
			zap.Strings("missing", errorScopes))
		return nil, fmt.Errorf("missing required scopes for app %s: %s", appID, strings.Join(errorScopes, ", "))
	}

	ctx = context.WithValue(ctx, oauth2.HTTPClient, s.client)

	token, err := s.exchanger(ctx, cfg, code, verifier)
	if err != nil {
		s.logger.Error("token exchange error", zap.Error(err))
		return nil, err
	}

	res := &TokenResult{
		AccessToken: token.AccessToken,
		TokenType:   token.TokenType,
		Expires: ExpiryInfo{
			Milli: token.Expiry.UnixNano() / 1000000,
			Fmt:   token.Expiry.Format(time.RFC3339),
		},
		Scopes: scopes,
	}

	if app.IncludeRefreshToken {
		res.RefreshToken = token.RefreshToken
	}

	id, email, name := getIdentifiersFromToken(ctx, s.client, token, app)
	res.User = UserInfo{ID: id, Email: email, Name: name}

	return res, nil
}

func (s *OAuthServiceImpl) Refresh(ctx context.Context, appID, refreshToken string) (*RefreshResult, error) {
	if appID == "" || refreshToken == "" {
		s.logger.Error("refresh invalid parameters", zap.String("appID", appID))
		return nil, fmt.Errorf("required data not provided")
	}

	sourcer, ok := s.sourcers[appID]
	if !ok {
		return nil, fmt.Errorf("app not found: %s", appID)
	}

	ctx = context.WithValue(ctx, oauth2.HTTPClient, s.client)

	tokenSource := sourcer.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken})
	token, err := tokenSource.Token()
	if err != nil {
		s.logger.Error("refresh error", zap.Error(err))
		return nil, err
	}

	return &RefreshResult{
		AccessToken: token.AccessToken,
		Expires: ExpiryInfo{
			Milli: token.Expiry.UnixNano() / 1000000,
			Fmt:   token.Expiry.Format(time.RFC3339),
		},
	}, nil
}

func (s *OAuthServiceImpl) List() []AppInfo {
	result := make([]AppInfo, 0, len(s.apps))
	for _, app := range s.apps {
		result = append(result, AppConfigToInfo(app))
	}
	return result
}

func (s *OAuthServiceImpl) buildHTTPClient() *http.Client {
	return &http.Client{
		Timeout: s.httpCfg.Timeout,
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			ResponseHeaderTimeout: s.httpCfg.ResponseHeaderTimeout,
			DialContext: (&net.Dialer{
				Timeout:   s.httpCfg.Timeout,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			IdleConnTimeout:     s.httpCfg.IdleConnTimeout,
			TLSHandshakeTimeout: s.httpCfg.TLSHandshakeTimeout,
			TLSClientConfig: &tls.Config{
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: s.httpCfg.TLSInsecureSkipVerify, //nolint:gosec // user-configurable flag, default false
			},
		},
	}
}
