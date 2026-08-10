// Copyright (c) 2020-2026, The MathWorks, Inc. All rights reserved.

package proxy

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func tokenWithIDToken(claims map[string]any) *oauth2.Token {
	payload, _ := json.Marshal(claims) //nolint:errcheck
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256"}`))
	body := base64.RawURLEncoding.EncodeToString(payload)
	sig := base64.RawURLEncoding.EncodeToString([]byte("sig"))
	idToken := strings.Join([]string{header, body, sig}, ".")

	token := &oauth2.Token{AccessToken: "access"}
	return token.WithExtra(map[string]any{"id_token": idToken})
}

func TestGetIdentifiersFromToken_FromIDToken(t *testing.T) {
	r := require.New(t)
	app := fullTestApp("100")
	client := NewMockHTTPClient(t)

	token := tokenWithIDToken(map[string]any{
		"sub":   "user-123",
		"email": "user@test.com",
		"name":  "Test User",
	})

	id, email, name := getIdentifiersFromToken(context.Background(), client, token, app)
	r.Equal("user-123", id)
	r.Equal("user@test.com", email)
	r.Equal("Test User", name)
}

func TestGetIdentifiersFromToken_NameFallbackToEmail(t *testing.T) {
	r := require.New(t)
	app := fullTestApp("100")
	client := NewMockHTTPClient(t)

	token := tokenWithIDToken(map[string]any{
		"sub":   "user-123",
		"email": "user@test.com",
	})

	id, email, name := getIdentifiersFromToken(context.Background(), client, token, app)
	r.Equal("user-123", id)
	r.Equal("user@test.com", email)
	r.Equal("user@test.com", name)
}

func TestGetIdentifiersFromToken_FallsBackToAPI(t *testing.T) {
	r := require.New(t)
	app := fullTestApp("100")
	client := NewMockHTTPClient(t)

	token := &oauth2.Token{AccessToken: "access"}

	client.EXPECT().Do(mock.Anything).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(`{"sub":"api-id","email":"api@test.com","name":"API User"}`)),
	}, nil)

	id, email, name := getIdentifiersFromToken(context.Background(), client, token, app)
	r.Equal("api-id", id)
	r.Equal("api@test.com", email)
	r.Equal("API User", name)
}

func TestGetIdentifiersFromToken_IDTokenMissingEmail_FallsBackToAPI(t *testing.T) {
	r := require.New(t)
	app := fullTestApp("100")
	client := NewMockHTTPClient(t)

	token := tokenWithIDToken(map[string]any{
		"sub": "user-123",
	})

	client.EXPECT().Do(mock.Anything).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(`{"sub":"api-id","email":"api@test.com","name":"API User"}`)),
	}, nil)

	id, email, name := getIdentifiersFromToken(context.Background(), client, token, app)
	r.Equal("api-id", id)
	r.Equal("api@test.com", email)
	r.Equal("API User", name)
}

func TestGetAccountInfoFromOAuth_Success(t *testing.T) {
	r := require.New(t)
	client := NewMockHTTPClient(t)

	client.EXPECT().Do(mock.Anything).Run(func(req *http.Request) {
		r.Equal("Bearer test-token", req.Header.Get("Authorization"))
		r.Equal("test-host", req.Header.Get("Host"))
	}).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(`{"email":"u@test.com","name":"User","sub":"id123"}`)),
	}, nil)

	data, err := getAccountInfoFromOAuth(context.Background(), client, "test-token", "test-host", "https://example.com/userinfo", "email")
	r.NoError(err)
	r.Equal("u@test.com", data["email"])
	r.Equal("User", data["name"])
}

func TestGetAccountInfoFromOAuth_ListResponse(t *testing.T) {
	r := require.New(t)
	client := NewMockHTTPClient(t)

	client.EXPECT().Do(mock.Anything).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(`[{"email":true,"name":"Primary"},{"email":false,"name":"Secondary"}]`)),
	}, nil)

	data, err := getAccountInfoFromOAuth(context.Background(), client, "tok", "host", "https://example.com/info", "email")
	r.NoError(err)
	r.Equal("Primary", data["name"])
}

func TestGetAccountInfoFromOAuth_HTTPError(t *testing.T) {
	r := require.New(t)
	client := NewMockHTTPClient(t)

	client.EXPECT().Do(mock.Anything).Return(nil, fmt.Errorf("connection refused"))

	_, err := getAccountInfoFromOAuth(context.Background(), client, "tok", "host", "https://example.com/info", "email")
	r.Error(err)
	r.Contains(err.Error(), "connection refused")
}

func TestGetAccountInfoFromOAuth_InvalidJSON(t *testing.T) {
	r := require.New(t)
	client := NewMockHTTPClient(t)

	client.EXPECT().Do(mock.Anything).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(`not json`)),
	}, nil)

	_, err := getAccountInfoFromOAuth(context.Background(), client, "tok", "host", "https://example.com/info", "email")
	r.Error(err)
}

func TestMakeAPICallForAccountInfo_ErrorReturnsUnknown(t *testing.T) {
	r := require.New(t)
	client := NewMockHTTPClient(t)

	client.EXPECT().Do(mock.Anything).Return(nil, fmt.Errorf("network error"))

	app := fullTestApp("100")
	token := &oauth2.Token{AccessToken: "test"}
	id, email, name := makeAPICallForAccountInfo(context.Background(), client, token, app)
	r.Equal("unknown", id)
	r.Equal("unknown", email)
	r.Equal("unknown", name)
}
