// Copyright (c) 2020-2026, The MathWorks, Inc. All rights reserved.

package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mathworks-ref-arch/container-images/oauth2-auth-code-proxy/internal/admin"
	"github.com/mathworks-ref-arch/container-images/oauth2-auth-code-proxy/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubOAuthService struct{}

func (s *stubOAuthService) Authorize(appID, redirectURL string) (string, string, error) {
	return "https://auth.example.com/authorize?code=test", "verifier123", nil
}

func (s *stubOAuthService) Token(_ context.Context, code, state, verifier, scope string) (*domain.TokenResult, error) {
	return &domain.TokenResult{
		AccessToken: "access-token",
		Expires:     domain.ExpiryInfo{Milli: 9999999999999, Fmt: "2099-01-01T00:00:00Z"},
		User:        domain.UserInfo{ID: "1", Email: "test@test.com", Name: "Test"},
		Scopes:      []string{"email"},
	}, nil
}

func (s *stubOAuthService) Refresh(_ context.Context, appID, refreshToken string) (*domain.RefreshResult, error) {
	return &domain.RefreshResult{
		AccessToken: "refreshed-token",
		Expires:     domain.ExpiryInfo{Milli: 9999999999999, Fmt: "2099-01-01T00:00:00Z"},
	}, nil
}

func (s *stubOAuthService) List() []domain.AppInfo {
	return []domain.AppInfo{{ID: "100", Type: "oauth2", CloudProvider: "test"}}
}

func newTestServer() *Server {
	apps := map[string]domain.AppConfig{
		"100": {
			ID: "100", Type: "oauth2", CloudProvider: "test",
			ClientID: "client", ClientSecret: "secret",
			RedirectURI: "http://localhost/callback",
			Scopes:      []string{"email"},
			AuthURL:     "https://auth.example.com/authorize",
			TokenURL:    "https://auth.example.com/token",
			RevokeURL:   "https://auth.example.com/revoke",
			Host:        "auth.example.com",
			TestingURL:  "https://auth.example.com/test",
			NameKey:     "name", EmailKey: "email",
			IDTokenNameKey: "name", IDTokenEmailKey: "email",
			Description: "Test App", DescriptionCode: "test",
		},
	}
	info := domain.BuildInfo{Version: "test", BuildTime: "now", ModuleName: "oauth2-auth-code-proxy"}
	hc := admin.NewHealthChecker(apps, info, zap.NewNop())
	hc.InitURLs()
	return &Server{
		oauth:  &stubOAuthService{},
		apps:   apps,
		logger: zap.NewNop(),
		health: hc,
		listenCfg: domain.ListenConfig{
			RESTPort: "80", MWLegacyPort: "30045", MWLegacyEnabled: true,
		},
	}
}

func TestRESTHandler_Authorize(t *testing.T) {
	r := require.New(t)
	srv := newTestServer()

	body := `{"app_id":"100","redirect_url":"http://localhost:1234/callback"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/authorize", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.RESTHandler().ServeHTTP(w, req)

	r.Equal(http.StatusOK, w.Code)
	var resp map[string]string
	r.NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	r.Contains(resp["authorization_url"], "auth.example.com")
	r.NotEmpty(resp["code_verifier"])
}

func TestRESTHandler_Token(t *testing.T) {
	r := require.New(t)
	srv := newTestServer()

	body := `{"grant_type":"authorization_code","app_id":"100","code":"abc","state":"100-http://localhost:1234","code_verifier":"v","scope":"email"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/token", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.RESTHandler().ServeHTTP(w, req)

	r.Equal(http.StatusOK, w.Code)
	var resp map[string]any
	r.NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	r.Equal("access-token", resp["access_token"])
}

func TestRESTHandler_Apps(t *testing.T) {
	r := require.New(t)
	srv := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/v1/apps", nil)
	w := httptest.NewRecorder()
	srv.RESTHandler().ServeHTTP(w, req)

	r.Equal(http.StatusOK, w.Code)
	var resp map[string]any
	r.NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	apps := resp["apps"].([]any)
	r.Len(apps, 1)
}

func TestRESTHandler_AdminAlive(t *testing.T) {
	r := require.New(t)
	a := assert.New(t)
	srv := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/alive/ping", nil)
	w := httptest.NewRecorder()
	srv.RESTHandler().ServeHTTP(w, req)

	r.Equal(http.StatusOK, w.Code)
	var resp admin.AliveResponse
	r.NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	a.Equal("ping", resp.Echo)
	a.Equal("test", resp.Version)
}

func TestRESTHandler_Healthz(t *testing.T) {
	r := require.New(t)
	srv := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	srv.RESTHandler().ServeHTTP(w, req)

	r.Equal(http.StatusOK, w.Code)
	r.Equal("ok", w.Body.String())
}

func TestRESTHandler_Docs(t *testing.T) {
	r := require.New(t)
	a := assert.New(t)
	srv := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/v1/docs", nil)
	req.Host = "localhost:80"
	w := httptest.NewRecorder()
	srv.RESTHandler().ServeHTTP(w, req)

	r.Equal(http.StatusOK, w.Code)
	a.Contains(w.Body.String(), "swagger-ui")
}

func TestRESTHandler_OpenAPISpec(t *testing.T) {
	r := require.New(t)
	a := assert.New(t)
	srv := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/v1/openapi.json", nil)
	w := httptest.NewRecorder()
	srv.RESTHandler().ServeHTTP(w, req)

	r.Equal(http.StatusOK, w.Code)
	a.Equal("application/json", w.Header().Get("Content-Type"))
}

func TestMWLegacyHandler_API(t *testing.T) {
	r := require.New(t)
	srv := newTestServer()

	body := `{"mwtype":"oauth/InitRequest","appID":"100","redirectUrl":"http://localhost:1234/callback"}`
	req := httptest.NewRequest(http.MethodPost, "/oauth/v1/api", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.MWLegacyHandler().ServeHTTP(w, req)

	r.Equal(http.StatusOK, w.Code)
	var resp map[string]any
	r.NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	r.Equal("oauth/InitResponse", resp["mwtype"])
}

func TestMWLegacyHandler_AdminAlive(t *testing.T) {
	r := require.New(t)
	a := assert.New(t)
	srv := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/oauth/v1/admin/alive/echo-test", nil)
	w := httptest.NewRecorder()
	srv.MWLegacyHandler().ServeHTTP(w, req)

	r.Equal(http.StatusOK, w.Code)
	var resp admin.AliveResponse
	r.NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	a.Equal("echo-test", resp.Echo)
}

func TestMWLegacyHandler_Redirect(t *testing.T) {
	r := require.New(t)
	srv := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/oauth/v1/redirect?code=abc&state=100-http://localhost:1234/cb", nil)
	w := httptest.NewRecorder()
	srv.MWLegacyHandler().ServeHTTP(w, req)

	r.Equal(http.StatusOK, w.Code)
	r.Contains(w.Body.String(), "http://localhost:1234/cb")
}

func TestMWLegacyHandler_Token(t *testing.T) {
	r := require.New(t)
	srv := newTestServer()

	body := `{"mwtype":"oauth/TokenRequest","appID":"100","auth_code":"abc","state":"100-http://localhost:1234","pkce_verifier":"v","scope":"email"}`
	req := httptest.NewRequest(http.MethodPost, "/oauth/v1/api", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.MWLegacyHandler().ServeHTTP(w, req)

	r.Equal(http.StatusOK, w.Code)
	var resp map[string]any
	r.NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	r.Equal("oauth/TokenResponse", resp["mwtype"])
	r.Equal("access-token", resp["accessToken"])
}

func TestMWLegacyHandler_Refresh(t *testing.T) {
	r := require.New(t)
	srv := newTestServer()

	body := `{"mwtype":"oauth/RefreshRequest","appID":"100","refreshToken":"rt-123"}`
	req := httptest.NewRequest(http.MethodPost, "/oauth/v1/api", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.MWLegacyHandler().ServeHTTP(w, req)

	r.Equal(http.StatusOK, w.Code)
	var resp map[string]any
	r.NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	r.Equal("oauth/RefreshResponse", resp["mwtype"])
	r.Equal("refreshed-token", resp["accessToken"])
}

func TestMWLegacyHandler_List(t *testing.T) {
	r := require.New(t)
	srv := newTestServer()

	body := `{"mwtype":"oauth/ListRequest"}`
	req := httptest.NewRequest(http.MethodPost, "/oauth/v1/api", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.MWLegacyHandler().ServeHTTP(w, req)

	r.Equal(http.StatusOK, w.Code)
	var resp map[string]any
	r.NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	r.Equal("oauth/ListResponse", resp["mwtype"])
	apps := resp["apps"].([]any)
	r.Len(apps, 1)
}

func TestMWLegacyHandler_UnknownMWType(t *testing.T) {
	r := require.New(t)
	srv := newTestServer()

	body := `{"mwtype":"unknown/Type"}`
	req := httptest.NewRequest(http.MethodPost, "/oauth/v1/api", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.MWLegacyHandler().ServeHTTP(w, req)

	r.Equal(http.StatusOK, w.Code)
	var resp map[string]any
	r.NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	r.Equal("core/Fault", resp["mwtype"])
}

func TestRESTHandler_TokenRefresh(t *testing.T) {
	r := require.New(t)
	srv := newTestServer()

	body := `{"grant_type":"refresh_token","app_id":"100","refresh_token":"rt-123"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/token", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.RESTHandler().ServeHTTP(w, req)

	r.Equal(http.StatusOK, w.Code)
	var resp map[string]any
	r.NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	r.Equal("refreshed-token", resp["access_token"])
}

func TestRESTHandler_Readyz(t *testing.T) {
	r := require.New(t)
	srv := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	srv.RESTHandler().ServeHTTP(w, req)

	r.Equal(http.StatusOK, w.Code)
	r.Equal("ok", w.Body.String())
}

func TestSecurityHeaders(t *testing.T) {
	a := assert.New(t)
	srv := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	srv.RESTHandler().ServeHTTP(w, req)

	a.Equal("nosniff", w.Header().Get("X-Content-Type-Options"))
	a.Equal("DENY", w.Header().Get("X-Frame-Options"))
}
