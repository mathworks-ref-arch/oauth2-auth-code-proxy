// Copyright (c) 2020-2026, The MathWorks, Inc. All rights reserved.

package mwlegacy

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/mathworks-ref-arch/container-images/oauth2-auth-code-proxy/internal/domain"
	"go.uber.org/zap"
)

type handler struct {
	oauth  domain.OAuthService
	apps   map[string]domain.AppConfig
	logger *zap.Logger
}

func Register(mux *http.ServeMux, svc domain.OAuthService, apps map[string]domain.AppConfig, logger *zap.Logger) {
	h := &handler{oauth: svc, apps: apps, logger: logger}
	mux.HandleFunc("POST /oauth/v1/api", h.handleV1API)
}

type InitRequest struct {
	AppID       string `json:"appID"`
	RedirectURL string `json:"redirectUrl"`
}

type InitResponse struct {
	MWType   string `json:"mwtype"`
	URL      string `json:"url"`
	Verifier string `json:"pkce_verifier"`
}

type TokenRequest struct {
	AppID    string `json:"appID"`
	Code     string `json:"auth_code"`
	State    string `json:"state"`
	Scope    string `json:"scope"`
	Verifier string `json:"pkce_verifier"`
}

type TokenResponse struct {
	MWType       string   `json:"mwtype"`
	RefreshToken string   `json:"refreshToken"`
	AccessToken  string   `json:"accessToken"`
	User         User     `json:"user"`
	Expires      Expiry   `json:"expires"`
	Scopes       []string `json:"scopes"`
}

type User struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	ID    string `json:"id"`
}

type Expiry struct {
	Milli int64  `json:"milli"`
	Fmt   string `json:"fmt"`
}

type RefreshRequest struct {
	AppID        string `json:"appID"`
	RefreshToken string `json:"refreshToken"`
}

type RefreshResponse struct {
	MWType      string `json:"mwtype"`
	AccessToken string `json:"accessToken"`
	Expires     Expiry `json:"expires"`
}

type ListRequest struct{}

type ListResponse struct {
	MWType string           `json:"mwtype"`
	Apps   []domain.AppInfo `json:"apps"`
}

// @Summary		Dispatch OAuth2 operation
// @Description	Single endpoint that dispatches on the "mwtype" field in the JSON body.
// @Description	Supported types: oauth/InitRequest, oauth/TokenRequest, oauth/RefreshRequest, oauth/ListRequest.
// @Tags			oauth
// @Accept			json
// @Produce		json
// @Param			body	body		object	true	"JSON envelope with mwtype discriminator"
// @Success		200		{object}	InitResponse		"InitResponse (mwtype=oauth/InitRequest)"
// @Success		200		{object}	TokenResponse		"TokenResponse (mwtype=oauth/TokenRequest)"
// @Success		200		{object}	RefreshResponse		"RefreshResponse (mwtype=oauth/RefreshRequest)"
// @Success		200		{object}	ListResponse		"ListResponse (mwtype=oauth/ListRequest)"
// @Failure		200		{object}	domain.FaultResponse	"Fault on invalid input or unknown mwtype"
// @Router			/api [post]
func (h *handler) handleV1API(w http.ResponseWriter, r *http.Request) {
	var envelope map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
		domain.WriteFault(w, "error.parse", "invalid request body")
		return
	}

	var mwtype string
	if raw, ok := envelope["mwtype"]; ok {
		json.Unmarshal(raw, &mwtype) //nolint:errcheck
	}

	ctx := r.Context()
	switch mwtype {
	case "oauth/InitRequest":
		h.handleInit(w, envelope)
	case "oauth/TokenRequest":
		h.handleToken(ctx, w, envelope)
	case "oauth/RefreshRequest":
		h.handleRefresh(ctx, w, envelope)
	case "oauth/ListRequest":
		h.handleList(w)
	default:
		domain.WriteFault(w, "error.unknown", "unknown message type: "+mwtype)
	}
}

func decodeEnvelope[T any](envelope map[string]json.RawMessage) (T, error) {
	b, _ := json.Marshal(envelope) //nolint:errcheck
	var v T
	err := json.Unmarshal(b, &v)
	return v, err
}

func (h *handler) handleInit(w http.ResponseWriter, envelope map[string]json.RawMessage) {
	req, err := decodeEnvelope[InitRequest](envelope)
	if err != nil {
		domain.WriteFault(w, "error.init", "invalid init request")
		return
	}

	url, verifier, err := h.oauth.Authorize(req.AppID, req.RedirectURL)
	if err != nil {
		h.logger.Error("handleInit", zap.Error(err))
		domain.WriteFault(w, "error.init", "authorization failed")
		return
	}

	domain.WriteJSON(w, InitResponse{
		MWType:   "oauth/InitResponse",
		URL:      url,
		Verifier: verifier,
	})
}

func (h *handler) handleToken(ctx context.Context, w http.ResponseWriter, envelope map[string]json.RawMessage) {
	req, err := decodeEnvelope[TokenRequest](envelope)
	if err != nil {
		domain.WriteFault(w, "error.token", "invalid token request")
		return
	}

	res, err := h.oauth.Token(ctx, req.Code, req.State, req.Verifier, req.Scope)
	if err != nil {
		h.logger.Error("handleToken", zap.Error(err))
		domain.WriteFault(w, "error.token", "token exchange failed")
		return
	}

	domain.WriteJSON(w, TokenResponse{
		MWType:       "oauth/TokenResponse",
		RefreshToken: res.RefreshToken,
		AccessToken:  res.AccessToken,
		User:         User{Email: res.User.Email, Name: res.User.Name, ID: res.User.ID},
		Expires:      Expiry{Milli: res.Expires.Milli, Fmt: res.Expires.Fmt},
		Scopes:       res.Scopes,
	})
}

func (h *handler) handleRefresh(ctx context.Context, w http.ResponseWriter, envelope map[string]json.RawMessage) {
	req, err := decodeEnvelope[RefreshRequest](envelope)
	if err != nil {
		domain.WriteFault(w, "error.refresh", "invalid refresh request")
		return
	}

	res, err := h.oauth.Refresh(ctx, req.AppID, req.RefreshToken)
	if err != nil {
		h.logger.Error("handleRefresh", zap.Error(err))
		domain.WriteFault(w, "token.refresh", "token refresh failed")
		return
	}

	domain.WriteJSON(w, RefreshResponse{
		MWType:      "oauth/RefreshResponse",
		AccessToken: res.AccessToken,
		Expires:     Expiry{Milli: res.Expires.Milli, Fmt: res.Expires.Fmt},
	})
}

func (h *handler) handleList(w http.ResponseWriter) {
	apps := h.oauth.List()
	domain.WriteJSON(w, ListResponse{
		MWType: "oauth/ListResponse",
		Apps:   apps,
	})
}
