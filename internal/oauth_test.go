// Copyright (c) 2020-2026, The MathWorks, Inc. All rights reserved.

package proxy

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

func newTestOAuthService(t *testing.T) *OAuthServiceImpl {
	t.Helper()
	apps := map[string]AppConfig{"100": fullTestApp("100")}
	httpCfg := HTTPClientConfig{
		Timeout:               10 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
	}
	svc := NewOAuthService(apps, httpCfg, zap.NewNop())
	impl, ok := svc.(*OAuthServiceImpl)
	if !ok {
		t.Fatal("NewOAuthService did not return *OAuthServiceImpl")
	}
	return impl
}

func TestAuthorize_Success(t *testing.T) {
	r := require.New(t)
	svc := newTestOAuthService(t)

	url, verifier, err := svc.Authorize("100", "http://localhost:12345?state=matlabid&foo=bar")
	r.NoError(err)
	r.NotEmpty(url)
	r.NotEmpty(verifier)
	r.Contains(url, "accounts.google.com")
	r.Contains(url, "state=100-http")
}

func TestAuthorize_StateEncoding(t *testing.T) {
	r := require.New(t)
	svc := newTestOAuthService(t)

	url, _, err := svc.Authorize("100", "http://localhost:12345?state=foo")
	r.NoError(err)
	r.Contains(url, "state=100-http%3A%2F%2Flocalhost%3A12345%3Fstate%3Dfoo")
}

func TestAuthorize_AppNotFound(t *testing.T) {
	r := require.New(t)
	svc := newTestOAuthService(t)

	_, _, err := svc.Authorize("999", "http://localhost")
	r.Error(err)
	r.Contains(err.Error(), "app not found")
}

func TestAuthorize_NoRedirectURL(t *testing.T) {
	r := require.New(t)
	svc := newTestOAuthService(t)

	url, _, err := svc.Authorize("100", "")
	r.NoError(err)
	r.Contains(url, "state=100")
	r.NotContains(url, "state=100-")
}

func TestToken_MissingScopesReturnsError(t *testing.T) {
	r := require.New(t)
	svc := newTestOAuthService(t)

	_, err := svc.Token(context.Background(), "code123", "100-http://localhost:12345", "verifier", "email")
	r.Error(err)
	r.Contains(err.Error(), "missing required scopes")
}

func TestToken_AppIDFromState(t *testing.T) {
	r := require.New(t)
	svc := newTestOAuthService(t)

	mockClient := NewMockHTTPClient(t)
	mockClient.EXPECT().Do(mock.Anything).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(`{"email":"test@example.com","name":"Test User","sub":"123"}`)),
	}, nil)

	svc.exchanger = func(_ context.Context, cfg *oauth2.Config, _, _ string) (*oauth2.Token, error) {
		r.Equal("fake-client-id", cfg.ClientID)
		return &oauth2.Token{
			AccessToken: "test-access-token",
			Expiry:      time.Now().Add(1 * time.Hour),
		}, nil
	}
	svc.client = mockClient

	scope := strings.Join(svc.apps["100"].Scopes, " ")
	res, err := svc.Token(context.Background(), "code123", "100-http://localhost:12345", "verifier", scope)
	r.NoError(err)
	r.Equal("test-access-token", res.AccessToken)
	r.Equal("test@example.com", res.User.Email)
	r.Equal("Test User", res.User.Name)
}

func TestToken_ExchangeError(t *testing.T) {
	r := require.New(t)
	svc := newTestOAuthService(t)

	svc.exchanger = func(_ context.Context, _ *oauth2.Config, _, _ string) (*oauth2.Token, error) {
		return nil, &oauth2.RetrieveError{Response: &http.Response{StatusCode: 400}}
	}

	scope := strings.Join(svc.apps["100"].Scopes, " ")
	_, err := svc.Token(context.Background(), "badcode", "100-http://localhost", "verifier", scope)
	r.Error(err)
}

func TestRefresh_Success(t *testing.T) {
	r := require.New(t)
	svc := newTestOAuthService(t)

	expiry := time.Now().Add(1 * time.Hour)
	mockSourcer := NewMockTokenSourcer(t)
	mockTS := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: "refreshed-token",
		Expiry:      expiry,
	})
	mockSourcer.EXPECT().TokenSource(mock.Anything, mock.Anything).Return(mockTS)
	svc.sourcers["100"] = mockSourcer

	res, err := svc.Refresh(context.Background(), "100", "old-refresh-token")
	r.NoError(err)
	r.Equal("refreshed-token", res.AccessToken)
	r.Equal(expiry.UnixNano()/1000000, res.Expires.Milli)
}

func TestRefresh_EmptyParams(t *testing.T) {
	r := require.New(t)
	svc := newTestOAuthService(t)

	_, err := svc.Refresh(context.Background(), "", "some-token")
	r.Error(err)
	r.Contains(err.Error(), "required data not provided")

	_, err = svc.Refresh(context.Background(), "100", "")
	r.Error(err)
	r.Contains(err.Error(), "required data not provided")
}

func TestRefresh_AppNotFound(t *testing.T) {
	r := require.New(t)
	svc := newTestOAuthService(t)

	_, err := svc.Refresh(context.Background(), "999", "some-token")
	r.Error(err)
	r.Contains(err.Error(), "app not found")
}

func TestList_ReturnsAppInfo(t *testing.T) {
	r := require.New(t)
	svc := newTestOAuthService(t)

	apps := svc.List()
	r.Len(apps, 1)
	r.Equal("100", apps[0].ID)
	r.Equal("fake-client-id", apps[0].ClientID)
	r.Equal("google", apps[0].CloudProvider)
}

func TestBuildHTTPClient_InsecureSkipVerify(t *testing.T) {
	r := require.New(t)
	apps := map[string]AppConfig{"100": fullTestApp("100")}
	httpCfg := HTTPClientConfig{
		Timeout:               10 * time.Second,
		TLSInsecureSkipVerify: true,
	}
	svc, ok := NewOAuthService(apps, httpCfg, zap.NewNop()).(*OAuthServiceImpl)
	r.True(ok)
	client := svc.buildHTTPClient()
	transport, ok := client.Transport.(*http.Transport)
	r.True(ok)
	r.True(transport.TLSClientConfig.InsecureSkipVerify)
}

func TestBuildHTTPClient_SecureByDefault(t *testing.T) {
	r := require.New(t)
	apps := map[string]AppConfig{"100": fullTestApp("100")}
	httpCfg := HTTPClientConfig{
		Timeout: 10 * time.Second,
	}
	svc, ok := NewOAuthService(apps, httpCfg, zap.NewNop()).(*OAuthServiceImpl)
	r.True(ok)
	client := svc.buildHTTPClient()
	transport, ok := client.Transport.(*http.Transport)
	r.True(ok)
	r.False(transport.TLSClientConfig.InsecureSkipVerify)
}
