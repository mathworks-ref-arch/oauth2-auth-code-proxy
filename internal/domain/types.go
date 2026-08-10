// Copyright (c) 2020-2026, The MathWorks, Inc. All rights reserved.

package domain

import (
	"context"
	"net/http"
	"time"

	"golang.org/x/oauth2"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type TokenSourcer interface {
	TokenSource(ctx context.Context, t *oauth2.Token) oauth2.TokenSource
	Config() *oauth2.Config
}

type OAuthService interface {
	Authorize(appID, redirectURL string) (authURL, verifier string, err error)
	Token(ctx context.Context, code, state, verifier, scope string) (*TokenResult, error)
	Refresh(ctx context.Context, appID, refreshToken string) (*RefreshResult, error)
	List() []AppInfo
}

type TokenResult struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	Expires      ExpiryInfo
	User         UserInfo
	Scopes       []string
}

type RefreshResult struct {
	AccessToken string
	Expires     ExpiryInfo
}

type ExpiryInfo struct {
	Milli int64
	Fmt   string
}

type UserInfo struct {
	ID    string
	Email string
	Name  string
}

type AppConfig struct {
	ID                  string   `yaml:"id" json:"id"`
	Type                string   `yaml:"type" json:"type"`
	IncludeRefreshToken bool     `yaml:"includeRefreshToken" json:"includeRefreshToken"`
	CloudProvider       string   `yaml:"cloudProvider" json:"cloudProvider"`
	Description         string   `yaml:"description" json:"description"`
	DescriptionCode     string   `yaml:"descriptionCode" json:"descriptionCode"`
	ClientID            string   `yaml:"clientId" json:"clientId"`
	ClientSecret        string   `yaml:"clientSecret" json:"-"`
	RedirectURI         string   `yaml:"redirectUri" json:"redirectUri"`
	Scopes              []string `yaml:"scopes" json:"scopes"`
	AuthURL             string   `yaml:"authUrl" json:"authUrl"`
	TokenURL            string   `yaml:"tokenUrl" json:"tokenUrl"`
	RevokeURL           string   `yaml:"revokeUrl" json:"revokeUrl"`
	Host                string   `yaml:"host" json:"host"`
	TestingURL          string   `yaml:"testingUrl" json:"testingUrl"`
	NameKey             string   `yaml:"nameKey" json:"nameKey"`
	EmailKey            string   `yaml:"emailKey" json:"emailKey"`
	IDTokenNameKey      string   `yaml:"idTokenNameKey" json:"idTokenNameKey"`
	IDTokenEmailKey     string   `yaml:"idTokenEmailKey" json:"idTokenEmailKey"`
}

type AppInfo struct {
	ID                  string   `json:"id"`
	Type                string   `json:"type"`
	CloudProvider       string   `json:"cloudProvider"`
	Description         string   `json:"description"`
	DescriptionCode     string   `json:"descriptionCode"`
	ClientID            string   `json:"clientId"`
	RedirectURI         string   `json:"redirectUri"`
	IncludeRefreshToken bool     `json:"includeRefreshToken"`
	Scopes              []string `json:"scopes"`
}

func AppConfigToInfo(app AppConfig) AppInfo {
	return AppInfo{
		ID:                  app.ID,
		Type:                app.Type,
		CloudProvider:       app.CloudProvider,
		Description:         app.Description,
		DescriptionCode:     app.DescriptionCode,
		ClientID:            app.ClientID,
		RedirectURI:         app.RedirectURI,
		IncludeRefreshToken: app.IncludeRefreshToken,
		Scopes:              app.Scopes,
	}
}

type FaultResponse struct {
	MWType    string   `json:"mwtype"`
	FaultID   []string `json:"faultId"`
	MessageID string   `json:"messageId"`
}

type HTTPClientConfig struct {
	Timeout               time.Duration
	ResponseHeaderTimeout time.Duration
	IdleConnTimeout       time.Duration
	TLSHandshakeTimeout   time.Duration
	TLSInsecureSkipVerify bool
}

type ListenConfig struct {
	RESTPort        string
	MWLegacyPort    string
	MWLegacyEnabled bool
	TLSCert         string
	TLSKey          string
}

type BuildInfo struct {
	Version    string
	BuildTime  string
	ModuleName string
	GroupID    string
}
