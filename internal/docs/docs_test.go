// Copyright (c) 2020-2026, The MathWorks, Inc. All rights reserved.

package docs

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mathworks-ref-arch/container-images/oauth2-auth-code-proxy/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAPISpec_REST_FiltersPathsAndSchemas(t *testing.T) {
	r := require.New(t)
	a := assert.New(t)

	d := &docsHandler{prefix: "/v1", listenCfg: domain.ListenConfig{}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/openapi.json", d.handleOpenAPISpec)

	req := httptest.NewRequest(http.MethodGet, "/v1/openapi.json", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	r.Equal(http.StatusOK, w.Code)
	a.Equal("application/json", w.Header().Get("Content-Type"))

	var spec map[string]any
	r.NoError(json.Unmarshal(w.Body.Bytes(), &spec))

	paths, _ := spec["paths"].(map[string]any)
	a.Contains(paths, "/authorize")
	a.Contains(paths, "/token")
	a.Contains(paths, "/apps")
	a.Contains(paths, "/admin/alive/{echo}")
	a.NotContains(paths, "/api")
	a.NotContains(paths, "/redirect")

	servers, _ := spec["servers"].([]any)
	r.Len(servers, 1)
	srv0, _ := servers[0].(map[string]any)
	a.Equal("/v1", srv0["url"])

	components, _ := spec["components"].(map[string]any)
	schemas, _ := components["schemas"].(map[string]any)
	a.NotContains(schemas, "mwlegacy.InitResponse")
	a.NotContains(schemas, "domain.FaultResponse")
	a.NotContains(schemas, "mwlegacy.TokenResponse")
	a.Contains(schemas, "v1.AuthorizeRequest")
	a.Contains(schemas, "v1.AuthorizeResponse")
	a.Contains(schemas, "v1.TokenRequestBody")
	a.Contains(schemas, "v1.TokenResponseBody")
	a.Contains(schemas, "admin.AliveResponse")
}

func TestOpenAPISpec_Legacy_FiltersPathsAndSchemas(t *testing.T) {
	r := require.New(t)
	a := assert.New(t)

	d := &docsHandler{prefix: "/oauth/v1", listenCfg: domain.ListenConfig{}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /oauth/v1/openapi.json", d.handleOpenAPISpec)

	req := httptest.NewRequest(http.MethodGet, "/oauth/v1/openapi.json", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	r.Equal(http.StatusOK, w.Code)

	var spec map[string]any
	r.NoError(json.Unmarshal(w.Body.Bytes(), &spec))

	paths, _ := spec["paths"].(map[string]any)
	a.Contains(paths, "/api")
	a.Contains(paths, "/redirect")
	a.Contains(paths, "/admin/alive/{echo}")
	a.NotContains(paths, "/authorize")
	a.NotContains(paths, "/token")
	a.NotContains(paths, "/apps")

	servers, _ := spec["servers"].([]any)
	r.Len(servers, 1)
	srv0, _ := servers[0].(map[string]any)
	a.Equal("/oauth/v1", srv0["url"])

	components, _ := spec["components"].(map[string]any)
	schemas, _ := components["schemas"].(map[string]any)
	a.NotContains(schemas, "v1.AuthorizeRequest")
	a.NotContains(schemas, "v1.AuthorizeResponse")
	a.NotContains(schemas, "v1.TokenRequestBody")
	a.NotContains(schemas, "v1.TokenResponseBody")
	a.Contains(schemas, "mwlegacy.InitResponse")
	a.Contains(schemas, "domain.FaultResponse")
	a.Contains(schemas, "admin.AliveResponse")
}

func TestDocs_REST_MWLegacyEnabled_ShowsCrossLink(t *testing.T) {
	r := require.New(t)
	a := assert.New(t)

	d := &docsHandler{prefix: "/v1", listenCfg: domain.ListenConfig{
		RESTPort: "80", MWLegacyPort: "30045", MWLegacyEnabled: true,
	}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/docs", d.handleDocs)

	req := httptest.NewRequest(http.MethodGet, "/v1/docs", nil)
	req.Host = "myhost.example.com:80"
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	r.Equal(http.StatusOK, w.Code)
	a.Contains(w.Body.String(), "swagger-ui")
	a.Contains(w.Body.String(), "http://myhost.example.com:30045/oauth/v1/docs")
	a.Contains(w.Body.String(), "Legacy API docs")
}

func TestDocs_BetaDisabled_NoCrossLink(t *testing.T) {
	r := require.New(t)
	a := assert.New(t)

	d := &docsHandler{prefix: "/v1", listenCfg: domain.ListenConfig{
		RESTPort: "80", MWLegacyPort: "30045", MWLegacyEnabled: false,
	}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/docs", d.handleDocs)

	req := httptest.NewRequest(http.MethodGet, "/v1/docs", nil)
	req.Host = "localhost:80"
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	r.Equal(http.StatusOK, w.Code)
	a.NotContains(w.Body.String(), "<nav")
	a.Contains(w.Body.String(), "swagger-ui")
}

func TestFilterSpecByTags_PrunesSchemas(t *testing.T) {
	r := require.New(t)
	a := assert.New(t)

	raw := []byte(`{
		"paths": {
			"/foo": {"get": {"tags": ["v1"], "responses": {"200": {"content": {"application/json": {"schema": {"$ref": "#/components/schemas/FooResp"}}}}}}}
		},
		"components": {
			"schemas": {
				"FooResp": {"type": "object", "properties": {"nested": {"$ref": "#/components/schemas/Nested"}}},
				"Nested": {"type": "object"},
				"Unused": {"type": "object"}
			}
		}
	}`)

	result := filterSpecByTags(raw, map[string]bool{"v1": true})
	var doc map[string]any
	r.NoError(json.Unmarshal(result, &doc))

	components, _ := doc["components"].(map[string]any)
	schemas, _ := components["schemas"].(map[string]any)
	a.Contains(schemas, "FooResp")
	a.Contains(schemas, "Nested")
	a.NotContains(schemas, "Unused")
}
