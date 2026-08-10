// Copyright (c) 2020-2026, The MathWorks, Inc. All rights reserved.

package v1

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/mathworks-ref-arch/container-images/oauth2-auth-code-proxy/internal/domain"
	"go.uber.org/zap"
)

func expiresInSeconds(milli int64) int64 {
	v := (milli - time.Now().UnixMilli()) / 1000
	if v < 0 {
		return 0
	}
	return v
}

type handler struct {
	oauth  domain.OAuthService
	apps   map[string]domain.AppConfig
	logger *zap.Logger
}

func Register(mux *http.ServeMux, svc domain.OAuthService, apps map[string]domain.AppConfig, logger *zap.Logger) {
	h := &handler{oauth: svc, apps: apps, logger: logger}
	mux.HandleFunc("POST /v1/authorize", h.handleAuthorize)
	mux.HandleFunc("POST /v1/token", h.handleToken)
	mux.HandleFunc("GET /v1/apps", h.handleApps)
}

type AuthorizeRequest struct {
	AppID       string `json:"app_id"`
	RedirectURL string `json:"redirect_url"`
}

type AuthorizeResponse struct {
	AuthorizationURL string `json:"authorization_url"`
	CodeVerifier     string `json:"code_verifier"`
}

type TokenRequestBody struct {
	GrantType    string `json:"grant_type"`
	AppID        string `json:"app_id"`
	Code         string `json:"code"`
	State        string `json:"state"`
	CodeVerifier string `json:"code_verifier"`
	Scope        string `json:"scope"`
	RefreshToken string `json:"refresh_token"`
}

type TokenResponseBody struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token,omitempty"`
	TokenType    string   `json:"token_type"`
	ExpiresIn    int64    `json:"expires_in"`
	ExpiresAt    string   `json:"expires_at"`
	Scope        []string `json:"scope,omitempty"`
	User         *User    `json:"user,omitempty"`
}

type User struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	ID    string `json:"id"`
}

type AppsResponse struct {
	Apps []domain.AppInfo `json:"apps"`
}

type ErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// @Summary		Generate authorization URL
// @Description	Creates an OAuth2 authorization URL with PKCE code challenge for the specified app.
// @Tags			v1
// @Accept			json
// @Produce		json
// @Param			body	body		AuthorizeRequest	true	"Authorization request"
// @Success		200		{object}	AuthorizeResponse
// @Failure		400		{object}	ErrorResponse
// @Router			/authorize [post]
func (h *handler) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	var req AuthorizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	if req.AppID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "app_id is required")
		return
	}
	if req.RedirectURL == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "redirect_url is required")
		return
	}

	url, verifier, err := h.oauth.Authorize(req.AppID, req.RedirectURL)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	domain.WriteJSON(w, AuthorizeResponse{
		AuthorizationURL: url,
		CodeVerifier:     verifier,
	})
}

// @Summary		Token exchange
// @Description	Exchange an authorization code for tokens (grant_type=authorization_code)
// @Description	or refresh an access token (grant_type=refresh_token).
// @Tags			v1
// @Accept			json
// @Produce		json
// @Param			body	body		TokenRequestBody	true	"Token request"
// @Success		200		{object}	TokenResponseBody
// @Failure		400		{object}	ErrorResponse
// @Router			/token [post]
func (h *handler) handleToken(w http.ResponseWriter, r *http.Request) {
	var req TokenRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	ctx := r.Context()
	switch req.GrantType {
	case "authorization_code":
		h.handleTokenExchange(ctx, w, req)
	case "refresh_token":
		h.handleTokenRefresh(ctx, w, req)
	default:
		writeError(w, http.StatusBadRequest, "unsupported_grant_type", "grant_type must be authorization_code or refresh_token")
	}
}

func (h *handler) handleTokenExchange(ctx context.Context, w http.ResponseWriter, req TokenRequestBody) {
	if req.AppID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "app_id is required")
		return
	}
	if req.Code == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "code is required")
		return
	}

	res, err := h.oauth.Token(ctx, req.Code, req.State, req.CodeVerifier, req.Scope)
	if err != nil {
		h.logger.Error("token exchange failed", zap.Error(err))
		writeError(w, http.StatusBadRequest, "invalid_grant", "token exchange failed")
		return
	}

	domain.WriteJSON(w, TokenResponseBody{
		AccessToken:  res.AccessToken,
		RefreshToken: res.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    expiresInSeconds(res.Expires.Milli),
		ExpiresAt:    res.Expires.Fmt,
		Scope:        res.Scopes,
		User:         &User{Email: res.User.Email, Name: res.User.Name, ID: res.User.ID},
	})
}

func (h *handler) handleTokenRefresh(ctx context.Context, w http.ResponseWriter, req TokenRequestBody) {
	if req.AppID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "app_id is required")
		return
	}
	if req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "refresh_token is required")
		return
	}

	res, err := h.oauth.Refresh(ctx, req.AppID, req.RefreshToken)
	if err != nil {
		h.logger.Error("token refresh failed", zap.Error(err))
		writeError(w, http.StatusBadRequest, "invalid_grant", "token refresh failed")
		return
	}

	domain.WriteJSON(w, TokenResponseBody{
		AccessToken: res.AccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   expiresInSeconds(res.Expires.Milli),
		ExpiresAt:   res.Expires.Fmt,
	})
}

// @Summary		List configured apps
// @Description	Returns all configured OAuth applications with public fields only (no secrets).
// @Tags			v1
// @Produce		json
// @Success		200	{object}	AppsResponse
// @Router			/apps [get]
func (h *handler) handleApps(w http.ResponseWriter, _ *http.Request) {
	domain.WriteJSON(w, AppsResponse{Apps: h.oauth.List()})
}

func writeError(w http.ResponseWriter, status int, code, description string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{ //nolint:errcheck
		Error:            code,
		ErrorDescription: description,
	})
}
