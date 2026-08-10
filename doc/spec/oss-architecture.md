# OAuth2 Token Proxy — Architecture Specification

## 1. Overview

A standalone HTTP service that performs server-side OAuth2 authorization code flow with PKCE on behalf of desktop clients. Desktop applications authenticate with third-party cloud providers (Google, Azure, etc.) without embedding client secrets in distributed binaries.

**Problem:** Native desktop applications cannot safely store OAuth2 client secrets. Embedding secrets in binaries exposes them to extraction. The authorization code grant (RFC 6749 Section 4.1) requires a confidential client to exchange codes for tokens.

**Solution:** This proxy holds client secrets server-side and exposes a REST API that desktop clients call to initiate authorization, exchange codes for tokens, and refresh access tokens. PKCE (RFC 7636) binds the authorization request to the token exchange, preventing interception attacks even without a client secret on the desktop side.

### Standards

| Standard | Usage |
|----------|-------|
| RFC 6749 Section 4.1 | Authorization Code Grant flow |
| RFC 7636 | Proof Key for Code Exchange (S256 method) |
| OpenID Connect | Userinfo endpoint for identity claims |
| OpenAPI 3.1.0 | Machine-readable API specification |
| RFC 6749 Section 5.2 | Error response format (REST API) |

## 2. Standards Compliance

### RFC 6749 — Authorization Code Grant

The `/v1/authorize` endpoint generates an authorization URL with all required parameters (client_id, redirect_uri, scope, state, response_type=code). The `/v1/token` endpoint exchanges the authorization code for access and refresh tokens using the client secret stored server-side.

### RFC 7636 — PKCE

Every authorization request includes a PKCE code challenge using the S256 method:
1. Service generates a cryptographically random `code_verifier`
2. Computes `code_challenge = BASE64URL(SHA256(code_verifier))`
3. Includes `code_challenge` and `code_challenge_method=S256` in the authorization URL
4. Returns `code_verifier` to the desktop client
5. Token exchange includes `code_verifier` for provider validation

The plaintext method is not supported.

### OpenID Connect

After token exchange, the service calls the provider's userinfo endpoint (configured as `testingUrl`) to retrieve identity claims (email, name). Claims can also be extracted from the ID token JWT using configurable keys.

### Error Codes (REST API)

The REST API uses standard OAuth2 error codes per Section 5.2:
- `invalid_request` — missing or malformed parameters
- `invalid_grant` — code exchange or refresh failure
- `unsupported_grant_type` — must be `authorization_code` or `refresh_token`

## 3. Architecture

### Package Layout

```
main.go                     Entry point: delegates to app.Run()
internal/
  app/                      Composition layer — config, server, dual-port listener
  domain/                   Shared interfaces and types (DAG bottom, no imports up)
  v1/                       REST API handlers (POST /v1/authorize, /v1/token, GET /v1/apps)
  mwlegacy/                 Compatibility API (POST /oauth/v1/api, mwtype dispatch)
  admin/                    Health check + connectivity diagnostics
  redirect/                 OAuth2 callback page (browser redirect back to desktop client)
  docs/                     Swagger UI + OpenAPI spec serving with tag-based filtering
  (service layer)           OAuth service, JWT parsing, user info, config loading
```

### Import Graph

All arrows point down. No cycles.

```
app/ ──> v1/, mwlegacy/, admin/, redirect/, docs/, domain/, (service layer)
v1/  ──> domain/
mwlegacy/ ──> domain/
admin/ ──> domain/
redirect/ ──> domain/
docs/ ──> domain/
(service layer) ──> domain/
```

### Handler Registration

Each sub-package exports a `Register(mux *http.ServeMux, ...)` function. The `app` package composes them into two independent handlers:

```go
func (s *Server) RESTHandler() http.Handler {
    mux := http.NewServeMux()
    v1.Register(mux, s.oauth, s.apps, s.logger)
    admin.Register(mux, "/v1", s.health)
    redirect.Register(mux, "/v1", s.apps)
    docs.Register(mux, "/v1", s.listenCfg)
    mux.HandleFunc("GET /healthz", s.handleHealthz)
    mux.HandleFunc("GET /readyz", s.handleReadyz)
    return s.middleware(mux)
}
```

### Core Interface

```go
type OAuthService interface {
    Authorize(appID, redirectURL string) (authURL, verifier string, err error)
    Token(code, state, verifier, scope string) (*TokenResult, error)
    Refresh(appID, refreshToken string) (*RefreshResult, error)
    List() []AppInfo
}
```

### Dual-Port Serving

| Handler | Default Port | Description |
|---------|-------------|-------------|
| RESTHandler | 80 | Standards-compliant REST API |
| MWLegacyHandler (deprecated) | 30045 | Compatibility with legacy API |

Both share identical middleware (security headers, panic recovery, request logging, 1MB body limit).

**Single-port mode:** When `--mwlegacy-enabled=false`, only the REST handler starts. It binds to `--port` (default 30045), not `--rest-port`. This makes `--port` the universal "listen port" flag regardless of mode.

## 4. HTTP API Reference

### REST API (primary port, default 80)

#### POST /v1/authorize

Generate an OAuth2 authorization URL with PKCE.

Request:
```json
{
  "app_id": "100",
  "redirect_url": "http://localhost:31337/callback"
}
```

Response (200):
```json
{
  "authorization_url": "https://accounts.google.com/o/oauth2/auth?client_id=...&code_challenge=...&state=100-http://localhost:31337/callback",
  "code_verifier": "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
}
```

#### POST /v1/token

Exchange authorization code or refresh a token.

**Authorization code exchange:**
```json
{
  "grant_type": "authorization_code",
  "app_id": "100",
  "code": "4/0AeanS0a...",
  "state": "100-http://localhost:31337/callback",
  "code_verifier": "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk",
  "scope": "email https://www.googleapis.com/auth/drive.readonly"
}
```

Response (200):
```json
{
  "access_token": "ya29.a0AfH6SM...",
  "refresh_token": "1//0dx...",
  "token_type": "Bearer",
  "expires_in": 3599,
  "expires_at": "2026-05-29T15:00:00Z",
  "scope": ["email", "https://www.googleapis.com/auth/drive.readonly"],
  "user": {"email": "user@example.com", "name": "Jane Doe", "id": "12345"}
}
```

**Refresh token:**
```json
{
  "grant_type": "refresh_token",
  "app_id": "100",
  "refresh_token": "1//0dx..."
}
```

Response (200):
```json
{
  "access_token": "ya29.a0AfH6SM...",
  "token_type": "Bearer",
  "expires_in": 3599,
  "expires_at": "2026-05-29T15:00:00Z"
}
```

#### GET /v1/apps

List configured OAuth applications (public fields only, no secrets).

Response (200):
```json
{
  "apps": [{
    "id": "100",
    "type": "sheets",
    "cloudProvider": "google",
    "description": "Read Google Sheets",
    "clientId": "714814256765-...apps.googleusercontent.com",
    "redirectUri": "https://localhost:30045/oauth/v1/redirect",
    "includeRefreshToken": true,
    "scopes": ["https://www.googleapis.com/auth/drive.readonly", "email"]
  }]
}
```

#### Error Response (REST API)

```json
{
  "error": "invalid_request",
  "error_description": "app_id is required"
}
```

HTTP status codes: 400 for client errors. Error codes follow RFC 6749 Section 5.2.

### Compatibility API (secondary port, default 30045)

#### POST /oauth/v1/api

Single endpoint dispatching on the `mwtype` field in the JSON body.

**Supported mwtype values:**

| mwtype | Operation | Equivalent REST endpoint |
|--------|-----------|------------------------|
| `oauth/InitRequest` | Generate auth URL | POST /v1/authorize |
| `oauth/TokenRequest` | Exchange auth code | POST /v1/token (grant_type=authorization_code) |
| `oauth/RefreshRequest` | Refresh token | POST /v1/token (grant_type=refresh_token) |
| `oauth/ListRequest` | List apps | GET /v1/apps |

**Example — Init:**
```json
{"mwtype": "oauth/InitRequest", "appID": "100", "redirectUrl": "http://localhost:31337/callback"}
```

Response:
```json
{"mwtype": "oauth/InitResponse", "url": "https://accounts.google.com/...", "pkce_verifier": "..."}
```

**Example — Token:**
```json
{"mwtype": "oauth/TokenRequest", "appID": "100", "auth_code": "4/0AeanS0a...", "state": "100-http://localhost:31337/callback", "pkce_verifier": "...", "scope": "email https://..."}
```

Response:
```json
{
  "mwtype": "oauth/TokenResponse",
  "accessToken": "ya29...",
  "refreshToken": "1//0dx...",
  "user": {"email": "user@example.com", "name": "Jane Doe", "id": "12345"},
  "expires": {"milli": 1748530800000, "fmt": "2026-05-29T15:00:00Z"},
  "scopes": ["email", "https://www.googleapis.com/auth/drive.readonly"]
}
```

**Error Response (Compatibility API):**
```json
{"mwtype": "core/Fault", "faultId": ["error.token"], "messageId": "missing required scopes..."}
```

Errors are always returned with HTTP 200 status. Clients must inspect the `mwtype` field to detect faults.

### Admin

#### GET {prefix}/admin/alive/{echo}

Available at both `/v1/admin/alive/{echo}` and `/oauth/v1/admin/alive/{echo}`.

Returns service version, build info, HTTP proxy configuration, and per-URL connectivity diagnostics for all configured OAuth provider endpoints.

Response (200):
```json
{
  "echo": "ping",
  "version": "1.15.0",
  "artifactId": "github.com/.../oauth2-auth-code-proxy",
  "groupId": "",
  "buildTimestamp": "2026-05-29T10:00:00Z",
  "proxy": {"httpProxy": "", "httpsProxy": "", "noProxy": ""},
  "connCheck": [{
    "url": "https://oauth2.googleapis.com",
    "viaProxy": false,
    "times": {"dnsLookup": "0.012s", "tcpConnection": "0.025s", "ping": "0.025s", "tlsHandshake": "0.045s", "httpResponseTime": "0.089s"},
    "status": {"success": true, "error": "", "lastChecked": "2026-05-29T10:00:00Z"}
  }]
}
```

### Redirect

#### GET {prefix}/redirect

OAuth2 provider callback page. The provider redirects the user's browser here after authorization. The page then redirects to the desktop client's local HTTP listener.

**Query parameters:**
- `code` — authorization code from the provider
- `state` — format: `{appID}-{redirectURL}` (split on first hyphen)

**Allowed redirect schemes:** `http`, `https`, `mw-matlab`

The page renders localized text (based on `Accept-Language`) and includes a JavaScript redirect with a clickable fallback link. The browser tab title is set to the app's `description` field if configured.

### Probes

| Path | Behavior |
|------|----------|
| GET /healthz | Always returns 200 "ok" |
| GET /readyz | Returns 200 after URL connectivity initialization completes; 503 before |

### Behavioral Notes

**State parameter format:** The state value is `{appID}-{redirectURL}`. It is split on the first hyphen, so app IDs must not contain hyphens. The desktop client receives this state from the OAuth provider callback and passes it back to `/v1/token`.

**Scope validation:** Token exchange verifies that all scopes configured in `creds.yaml` are present in the provider's response. If the provider returns fewer scopes (e.g., user denied a permission), the exchange fails with an error listing the missing scopes.

**Refresh token availability:** The `includeRefreshToken` field in the credentials file controls whether `access_type=offline` is included in the authorization URL and whether the refresh token appears in token responses. When false, only access tokens are issued.

**No CORS:** This service is designed for server-to-server or desktop-native callers. No CORS headers are set. Browser-based JavaScript clients are not supported.

## 5. OpenAPI Spec Strategy

### Generation

A single OpenAPI 3.1 spec is generated from swag annotations across all sub-packages:

```
swag init --v3.1 --generalInfo doc.go --dir ./internal \
    --parseDependency --parseInternal --outputTypes json
```

Schema names derive from Go package names: `v1.AuthorizeRequest`, `mwlegacy.InitResponse`, `admin.AliveResponse`, `domain.AppInfo`.

### Runtime Filtering

At startup, the embedded spec is filtered into two views:

| API surface | Included tags | Base path |
|-------------|--------------|-----------|
| REST | `v1`, `admin` | `/v1` |
| Compatibility | `oauth`, `admin` | `/oauth/v1` |

Unreachable schemas are pruned via BFS over `$ref` links, starting from paths in the filtered spec. Each filtered view gets `servers[0].url` set to its base path.

### Swagger UI

`GET {prefix}/docs` serves a Swagger UI page that loads the filtered spec from `{prefix}/openapi.json`. When the legacy API is enabled, a cross-link navigation bar connects the two docs pages.

Swagger UI assets are loaded from the unpkg.com CDN. The `/docs` page requires outbound internet access to render.

## 6. Configuration

### CLI Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--config` | Path to credentials YAML file | (required) |
| `--port` | Listen port; applies to REST when mwlegacy disabled, legacy when enabled | 30045 |
| `--rest-port` | REST API port (only used when mwlegacy is enabled) | 80 |
| `--mwlegacy-enabled` | Enable compatibility API on separate port | true |
| `--tls-cert` | TLS certificate path | (none — serves HTTP) |
| `--tls-key` | TLS private key path | (none — serves HTTP) |
| `--tls-insecure-skip-verify` | Skip TLS verification on outbound calls | false |

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `OAUTH_CREDS_FILE` | Credentials file path | (none) |
| `OAUTH_REST_PORT` | REST API port | 80 |
| `OAUTH_PORT` | Compatibility API port | 30045 |
| `OAUTH_MWLEGACY_ENABLED` | Enable compatibility API (true/false) | true |
| `OAUTH_TLS_CERT_FILE` | TLS certificate path | (none) |
| `OAUTH_TLS_KEY_FILE` | TLS private key path | (none) |
| `OAUTH_TLS_INSECURE_SKIP_VERIFY` | Skip outbound TLS verification (1/true) | false |

### Precedence

Configuration is loaded in order (last writer wins):

1. Deprecated `--set` arguments
2. Deprecated `MW_*` environment variables
3. Application file (legacy config path)
4. CLI flags (`--config`, `--port`, etc.)
5. `OAUTH_*` environment variables

Environment variables override CLI flags. This is intentional for container orchestration where env vars are the primary configuration mechanism.

### Constraints

- `--port` and `--rest-port` cannot be the same value when mwlegacy is enabled
- When `--mwlegacy-enabled=false`, `--port` is used for REST (ignoring `--rest-port`)

### Credentials File Format

```yaml
id: "100"
type: "sheets"
includeRefreshToken: true
cloudProvider: "google"
description: "Read Google Sheets"
descriptionCode: "google.sheets.read"
clientId: "714814...apps.googleusercontent.com"
clientSecret: "GOCSPX-..."
redirectUri: "https://localhost:30045/oauth/v1/redirect"
scopes:
  - "https://www.googleapis.com/auth/drive.readonly"
  - "email"
authUrl: "https://accounts.google.com/o/oauth2/auth"
tokenUrl: "https://oauth2.googleapis.com/token"
revokeUrl: "https://oauth2.googleapis.com/revoke"
host: "sheets.googleapis.com"
testingUrl: "https://openidconnect.googleapis.com/v1/userinfo"
nameKey: "name"
emailKey: "email"
idTokenNameKey: "name"
idTokenEmailKey: "email"
```

### Field Reference

| Field | Required | Description |
|-------|----------|-------------|
| `id` | Yes | App identifier (must be "100") |
| `type` | Yes | App type label (e.g., "sheets", "onedrive") |
| `clientId` | Yes | OAuth2 client ID |
| `clientSecret` | Yes | OAuth2 client secret (kept server-side) |
| `redirectUri` | Yes | Registered redirect URI |
| `scopes` | Yes | Required scopes (must include "email") |
| `authUrl` | Yes | Provider's authorization endpoint |
| `tokenUrl` | Yes | Provider's token endpoint |
| `includeRefreshToken` | No | Enable `access_type=offline` and refresh tokens (default: false) |
| `cloudProvider` | No | Provider label for client display |
| `description` | No | Human-readable name (also used as redirect page title) |
| `descriptionCode` | No | i18n message key for description |
| `revokeUrl` | No | Token revocation endpoint |
| `host` | No | Provider API hostname |
| `testingUrl` | No | Userinfo endpoint for identity claims AND health diagnostics |
| `nameKey` | No | JSON key for user's name in userinfo response |
| `emailKey` | No | JSON key for user's email in userinfo response |
| `idTokenNameKey` | No | JWT claim key for name (fallback if userinfo unavailable) |
| `idTokenEmailKey` | No | JWT claim key for email (fallback if userinfo unavailable) |

Currently only a single app configuration (ID "100") is supported per instance.

## 7. Security Model

### What Is Protected

- **Client secrets** are stored server-side only; never exposed in API responses
- **TLS >= 1.2** enforced on all outbound connections to OAuth providers
- **Security headers** on every response:
  - `X-Content-Type-Options: nosniff`
  - `X-Frame-Options: DENY`
- **Request body limit**: 1MB maximum

### What Is NOT Stored

- No token persistence — the service is stateless
- No session state — each request is independent
- No user data beyond the immediate response
- Tokens are returned to the caller and never cached

### Container Security

- `scratch` base image: no shell, no package manager, no OS utilities
- Static binary with embedded CA certificates only
- Image size approximately 15MB

### Redirect Validation

- **Scheme allowlist**: only `http`, `https`, and `mw-matlab` are permitted
- **Host required**: redirect URLs must have a non-empty host
- **State integrity**: the state parameter encodes the app ID and redirect URL; both are validated before redirect

## 8. Build & CI

### Local Build

```
make all        # fmt → build → test → lint → cyclo
make build      # produces ./oauth2-auth-code-proxy
make test       # race detector + 80% coverage gate
make swagger    # regenerate OpenAPI spec from annotations
make docker     # multi-stage Docker build
```

### LDFLAGS

Build injects version metadata into `internal/app`:

```
-X 'internal/app.Version=$(VERSION)'
-X 'internal/app.BuildTime=$(BUILD_TS)'
-X 'internal/app.ModuleName=$(MODULE)'
-X 'internal/app.GroupID=$(GROUP_ID)'
```

### Docker

Multi-stage build: `golang:1.26` builder → `scratch` runtime. Final image contains only the static binary and CA certificates. Approximately 15MB.

```
docker build --build-arg VERSION=1.15.0 -t oauth2-auth-code-proxy .
```

### CI Pipeline

| Check | Threshold |
|-------|-----------|
| Unit tests | `-race`, must pass |
| Coverage | 80% per-package (`-coverpkg=./...`) |
| Lint | golangci-lint v2, zero issues |
| Cyclomatic complexity | max 9 per function (`gocyclo -over 9`) |
| Formatting | `goimports` + `go fmt` |
| Swagger freshness | `make swagger && git diff --exit-code` |

## 9. Deployment

### Docker

```
docker run -d \
  -p 80:80 -p 30045:30045 \
  -v /path/to/creds.yaml:/creds.yaml:ro \
  -e OAUTH_CREDS_FILE=/creds.yaml \
  oauth2-auth-code-proxy
```

For TLS:
```
docker run -d \
  -p 443:80 -p 30045:30045 \
  -v /path/to/creds.yaml:/creds.yaml:ro \
  -v /path/to/certs:/certs:ro \
  -e OAUTH_CREDS_FILE=/creds.yaml \
  -e OAUTH_TLS_CERT_FILE=/certs/tls.crt \
  -e OAUTH_TLS_KEY_FILE=/certs/tls.key \
  oauth2-auth-code-proxy
```

### HTTP Proxy

The service respects `HTTP_PROXY`, `HTTPS_PROXY`, and `NO_PROXY` for outbound connections to OAuth providers. These affect outbound calls only, not inbound request handling.

### Health Probes

| Probe | Path | Behavior |
|-------|------|----------|
| Liveness | `/healthz` | Always 200 |
| Readiness | `/readyz` | 200 after URL connectivity initialization; 503 before |

The readiness probe waits until the service has performed initial connectivity checks against all configured provider endpoints.

### Graceful Shutdown

On SIGINT, the service initiates graceful shutdown with a 30-second timeout for in-flight requests.

**Known limitation:** Only SIGINT is handled. Kubernetes sends SIGTERM for pod termination. Deploy with a process manager or configure `shareProcessNamespace` if SIGTERM handling is required.

### Kubernetes

A Helm chart is available at `deploy/helm/` (out of scope for this specification).

## 10. Internationalization

### CLI Help

Help output language is determined by the `LANG` environment variable. Translations use `golang.org/x/text/message` with gotext-generated catalogs.

### Redirect Page

The OAuth2 callback page renders localized text based on the browser's `Accept-Language` header.

### Supported Locales

| Locale | Language |
|--------|----------|
| en-US | English (default) |
| fr-FR | French |

### External Dependencies

The Swagger UI docs page (`/docs`) loads assets from the unpkg.com CDN. It requires outbound internet access to render. The API endpoints themselves have no CDN dependency.

## 11. API Evolution

The compatibility API (`/oauth/v1/api`) is maintained but not enhanced. Bug fixes are applied to both API surfaces. New features and endpoints are implemented exclusively in the REST API (`/v1/`).

The compatibility API exists to support existing desktop clients that were built against the original envelope-based protocol. There is no planned removal date. Contributors should implement new functionality in the `v1/` package only.
