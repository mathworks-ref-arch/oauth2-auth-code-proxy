// Copyright (c) 2020-2026, The MathWorks, Inc. All rights reserved.

package docs

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/mathworks-ref-arch/container-images/oauth2-auth-code-proxy/internal/domain"
)

//go:embed swagger.json
var openapiSpec []byte

var (
	restSpec   []byte
	legacySpec []byte
	specOnce   sync.Once
)

func buildFilteredSpecs() {
	restSpec = setSpecServers(
		filterSpecByTags(openapiSpec, map[string]bool{"v1": true, "admin": true}),
		"/v1",
	)
	legacySpec = setSpecServers(
		filterSpecByTags(openapiSpec, map[string]bool{"oauth": true, "admin": true}),
		"/oauth/v1",
	)
}

func filterSpecByTags(raw []byte, allowedTags map[string]bool) []byte {
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return raw
	}

	paths, ok := doc["paths"].(map[string]any)
	if !ok {
		return raw
	}

	filtered := make(map[string]any)
	for path, methods := range paths {
		if fm := filterPathMethods(methods, allowedTags); fm != nil {
			filtered[path] = fm
		}
	}

	doc["paths"] = filtered
	pruneUnusedSchemas(doc)
	out, err := json.MarshalIndent(doc, "", "    ")
	if err != nil {
		return raw
	}
	return out
}

func filterPathMethods(methods any, allowedTags map[string]bool) map[string]any {
	methodMap, ok := methods.(map[string]any)
	if !ok {
		return nil
	}
	result := make(map[string]any)
	for method, op := range methodMap {
		if operationHasTag(op, allowedTags) {
			result[method] = op
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func operationHasTag(op any, allowedTags map[string]bool) bool {
	opMap, ok := op.(map[string]any)
	if !ok {
		return false
	}
	tags, _ := opMap["tags"].([]any)
	for _, t := range tags {
		if tag, ok := t.(string); ok && allowedTags[tag] {
			return true
		}
	}
	return false
}

func pruneUnusedSchemas(doc map[string]any) {
	components, ok := doc["components"].(map[string]any)
	if !ok {
		return
	}
	schemas, ok := components["schemas"].(map[string]any)
	if !ok {
		return
	}

	reachable := findReachableSchemas(doc["paths"], schemas)
	for name := range schemas {
		if !reachable[name] {
			delete(schemas, name)
		}
	}
}

func findReachableSchemas(paths any, schemas map[string]any) map[string]bool {
	reachable := map[string]bool{}
	queue := collectRefs(paths)
	for len(queue) > 0 {
		ref := queue[0]
		queue = queue[1:]
		if reachable[ref] {
			continue
		}
		reachable[ref] = true
		if schema, exists := schemas[ref]; exists {
			queue = append(queue, collectRefs(schema)...)
		}
	}
	return reachable
}

func collectRefs(v any) []string {
	var refs []string
	switch val := v.(type) {
	case map[string]any:
		if ref, ok := val["$ref"].(string); ok {
			const prefix = "#/components/schemas/"
			if len(ref) > len(prefix) && ref[:len(prefix)] == prefix {
				refs = append(refs, ref[len(prefix):])
			}
		}
		for _, child := range val {
			refs = append(refs, collectRefs(child)...)
		}
	case []any:
		for _, item := range val {
			refs = append(refs, collectRefs(item)...)
		}
	}
	return refs
}

func setSpecServers(spec []byte, basePath string) []byte {
	var doc map[string]any
	if err := json.Unmarshal(spec, &doc); err != nil {
		return spec
	}
	doc["servers"] = []map[string]string{{"url": basePath}}
	out, err := json.MarshalIndent(doc, "", "    ")
	if err != nil {
		return spec
	}
	return out
}

type docsHandler struct {
	prefix    string
	listenCfg domain.ListenConfig
}

func Register(mux *http.ServeMux, prefix string, cfg domain.ListenConfig) {
	d := &docsHandler{prefix: prefix, listenCfg: cfg}
	mux.HandleFunc("GET "+prefix+"/openapi.json", d.handleOpenAPISpec)
	mux.HandleFunc("GET "+prefix+"/docs", d.handleDocs)
}

func (d *docsHandler) handleOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	specOnce.Do(buildFilteredSpecs)

	var spec []byte
	if strings.HasPrefix(d.prefix, "/v1") {
		spec = restSpec
	} else {
		spec = legacySpec
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(http.StatusOK)
	w.Write(spec) //nolint:errcheck
}

const swaggerUITemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>OAuth2 Token Proxy — API Docs</title>
<link rel="icon" href="favicon.ico" type="image/x-icon">
<link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
<link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.css">
</head>
<body>
%s
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
<script>
SwaggerUIBundle({
  url: window.location.origin + window.location.pathname.replace("/docs", "/openapi.json"),
  dom_id: "#swagger-ui",
  deepLinking: true,
  presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
  plugins: [SwaggerUIBundle.plugins.DownloadUrl],
  layout: "StandaloneLayout"
});
</script>
</body>
</html>`

func (d *docsHandler) handleDocs(w http.ResponseWriter, r *http.Request) {
	link := d.buildDocsLink(r)
	html := fmt.Sprintf(swaggerUITemplate, link)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src https://unpkg.com; style-src https://unpkg.com; img-src 'self' data:; connect-src 'self'")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html)) //nolint:errcheck
}

func (d *docsHandler) buildDocsLink(r *http.Request) string {
	if !d.listenCfg.MWLegacyEnabled() {
		return ""
	}

	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}

	host := r.Host
	hostname := host
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		hostname = host[:idx]
	}

	restPort := d.listenCfg.RESTPort
	legacyPort := d.listenCfg.MWLegacyPort

	isREST := strings.HasPrefix(d.prefix, "/v1")
	if isREST {
		otherURL := fmt.Sprintf("%s://%s:%s/oauth/v1/docs", scheme, hostname, legacyPort)
		return fmt.Sprintf(`<nav style="padding:0.5rem 1rem;background:#f8f8f8;border-bottom:1px solid #ddd;font-family:sans-serif;font-size:0.9rem;">Also available: <a href="%s">Legacy API docs</a></nav>`, otherURL)
	}

	otherURL := fmt.Sprintf("%s://%s:%s/v1/docs", scheme, hostname, restPort)
	return fmt.Sprintf(`<nav style="padding:0.5rem 1rem;background:#f8f8f8;border-bottom:1px solid #ddd;font-family:sans-serif;font-size:0.9rem;">Also available: <a href="%s">REST API (v1) docs</a></nav>`, otherURL)
}
