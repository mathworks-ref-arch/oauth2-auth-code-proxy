// Copyright (c) 2020-2026, The MathWorks, Inc. All rights reserved.

package redirect

import (
	"embed"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/mathworks-ref-arch/container-images/oauth2-auth-code-proxy/internal/domain"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

//go:embed redirect.html
var redirectFS embed.FS

//go:embed redirect.js
var redirectJS []byte

type handler struct {
	apps map[string]domain.AppConfig
}

func Register(mux *http.ServeMux, prefix string, apps map[string]domain.AppConfig) {
	h := &handler{apps: apps}
	mux.HandleFunc("GET "+prefix+"/redirect", h.handleRedirect)
	mux.HandleFunc("GET "+prefix+"/redirect.js", handleRedirectJS)
}

func handleRedirectJS(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.WriteHeader(http.StatusOK)
	w.Write(redirectJS) //nolint:errcheck
}

const redirectOverridePath = "/etc/oauth2-auth-code-proxy/redirect.html"

var readFileFunc = os.ReadFile

var allowedRedirectSchemes = map[string]bool{
	"http":      true,
	"https":     true,
	"mw-matlab": true,
}

func isAllowedRedirectScheme(scheme string) bool {
	return allowedRedirectSchemes[scheme]
}

type redirectData struct {
	URL       template.URL
	Lang      string
	Title     string
	Complete  string
	ClickHere string
}

var supportedLanguages = language.NewMatcher([]language.Tag{
	language.AmericanEnglish,
	language.French,
})

func matchAcceptLanguage(header string) language.Tag {
	tags, _, _ := language.ParseAcceptLanguage(header)
	tag, _, _ := supportedLanguages.Match(tags...)
	return tag
}

var (
	redirectTmpl     *template.Template
	redirectTmplOnce sync.Once
)

func loadRedirectTemplate() *template.Template {
	redirectTmplOnce.Do(func() {
		if data, err := readFileFunc(redirectOverridePath); err == nil {
			if tmpl, err := template.New("redirect.html").Parse(string(data)); err == nil {
				redirectTmpl = tmpl
				return
			}
		}
		redirectTmpl = template.Must(template.ParseFS(redirectFS, "redirect.html"))
	})
	return redirectTmpl
}

// @Summary		OAuth2 redirect callback
// @Description	Receives authorization code and state from the OAuth2 provider,
// @Description	then redirects the browser back to the desktop client's local HTTP listener.
// @Tags			oauth
// @Produce		html
// @Param			code	query		string	true	"Authorization code from OAuth2 provider"
// @Param			state	query		string	true	"State containing appID and client redirect URL (format: appID-redirectURL)"
// @Success		200		{string}	string				"HTML page that redirects the browser"
// @Failure		200		{object}	domain.FaultResponse	"Fault on missing or invalid parameters"
// @Router			/redirect [get]
func (h *handler) handleRedirect(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if state == "" || code == "" {
		domain.WriteFault(w, "StatusBadRequest", "state and code query params not found")
		return
	}

	appID, stateURL, faultMsg := parseRedirectState(state)
	if faultMsg != "" {
		domain.WriteFault(w, "StatusBadRequest", faultMsg)
		return
	}

	newURL := *r.URL
	newURL.Scheme = stateURL.Scheme
	newURL.Host = stateURL.Host
	newURL.Path = stateURL.Path

	tag := matchAcceptLanguage(r.Header.Get("Accept-Language"))
	p := message.NewPrinter(tag)

	title := p.Sprintf("OAuth2 Redirect")
	if app, ok := h.apps[appID]; ok && app.Description != "" {
		title = app.Description
	}

	data := redirectData{
		URL:       template.URL(newURL.String()),
		Lang:      tag.String(),
		Title:     title,
		Complete:  p.Sprintf("Authentication complete. You may close this tab."),
		ClickHere: p.Sprintf("Click here if not redirected automatically."),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy",
		"default-src 'none'; script-src 'self'; style-src 'unsafe-inline'; img-src 'self'; connect-src 'self'")
	w.WriteHeader(http.StatusOK)
	loadRedirectTemplate().Execute(w, data) //nolint:errcheck
}

func parseRedirectState(state string) (appID string, stateURL *url.URL, faultMsg string) {
	pieces := strings.SplitN(state, "-", 2)
	if len(pieces) != 2 {
		return "", nil, "app id and redirect url not found"
	}

	parsed, err := url.Parse(pieces[1])
	if err != nil {
		return "", nil, err.Error()
	}
	if parsed.Host == "" {
		return "", nil, "state redirect is invalid"
	}
	if !isAllowedRedirectScheme(parsed.Scheme) {
		return "", nil, "redirect scheme must be http, https, or mw-matlab"
	}
	return pieces[0], parsed, ""
}
