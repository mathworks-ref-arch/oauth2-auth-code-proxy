# Agent Instructions — oauth2-auth-code-proxy

## Service Purpose

OAuth2 token exchange proxy for MATLAB&reg; and MathWorks desktop clients. Handles the server-side OAuth2 authorization code flow (with PKCE) so desktop applications authenticate with third-party cloud providers without embedding client secrets.

## Architecture

- **Single binary**: `package main` at repo root. All application code in `internal/` package.
- **Entry point**: `main.go` -> `run()` pattern (testable without starting a process).
- **HTTP routing**: Go 1.22+ `http.ServeMux` method+path patterns.
- **Interfaces**: `OAuthService`, `HTTPClient`, `TokenSourcer` — all testable via mocks.
- **Mocks**: mockery v2 (`.mockery.yaml`), generated in-package as `*_test.go`.

## Key Files

| File | Responsibility |
|------|---------------|
| `doc.go` | OpenAPI metadata annotations (swag v2) |
| `config.go` | Config struct, CLI flags, env vars, creds.yaml loading |
| `oauth.go` | OAuthService interface + impl (Authorize, Token, Refresh, List) |
| `jwt.go` | Base64url JWT decode (no verification — claims extraction only) |
| `userinfo.go` | User identity extraction from id_token or userinfo endpoint |
| `handler.go` | POST /oauth/v1/api — mwtype JSON dispatch |
| `redirect.go` | GET /oauth/v1/redirect — OAuth2 callback handler |
| `health.go` | GET /oauth/v1/admin/alive — connectivity diagnostics |
| `server.go` | Server struct, mux setup, middleware |
| `run.go` | Entry wiring: config -> service -> server |

## Build & Test

The go.mod requires Go 1.26.3+. Local machines may not have the exact toolchain (and on Windows, `GOSUMDB=off` blocks automatic toolchain download), so **always build via Docker**:

```bash
make docker         # build Docker image (uses golang:1.26 — correct toolchain)
```

### Running locally for end-to-end testing

```bash
# 1. Create creds.yaml at repo root (git-ignored) — see README "Credentials File" section
# 2. Build and run:
make docker

docker run -d --name oauth2-proxy-test \
  -p 30045:30045 \
  -v "$(pwd)/creds.yaml:/etc/oauth2-auth-code-proxy/creds.yaml:ro" \
  -e OAUTH_CREDS_FILE=/etc/oauth2-auth-code-proxy/creds.yaml \
  oauth2-auth-code-proxy:latest

# 3. Verify:
curl http://localhost:30045/oauth/v1/admin/alive

# Stop/cleanup:
docker rm -f oauth2-proxy-test
```

### Windows (Git Bash / MSYS2) issues

Two problems affect `docker run` on Windows with Git Bash:

1. **MSYS path translation**: Git Bash auto-converts Unix-style paths to Windows paths. The volume mount target `/etc/oauth2-auth-code-proxy/creds.yaml` gets mangled to `C:/Program Files/Git/etc/oauth2-auth-code-proxy/creds.yaml` inside the container, causing a "file not found" error. Fix: prefix the command with `MSYS_NO_PATHCONV=1`.

2. **`make build` fails with GOSUMDB error**: The go.mod `toolchain` directive triggers a download of go1.26.3, which fails when `GOSUMDB=off` (common in corp environments). The error is: `verifying module: checksum database disabled by GOSUMDB=off`. This is why Docker is the canonical build path — the container image already has the correct toolchain.

Windows `docker run` command (note the env prefix):

```bash
MSYS_NO_PATHCONV=1 docker run -d --name oauth2-proxy-test \
  -p 30045:30045 \
  -v "$(pwd)/creds.yaml:/etc/oauth2-auth-code-proxy/creds.yaml:ro" \
  -e OAUTH_CREDS_FILE=/etc/oauth2-auth-code-proxy/creds.yaml \
  oauth2-auth-code-proxy:latest
```

If using PowerShell or cmd.exe instead of Git Bash, `MSYS_NO_PATHCONV` is not needed — the path translation issue is Git Bash–specific.

### Other make targets

These require Go 1.26.3+ locally or can be run inside the builder container:

```bash
make build          # produce oauth2-auth-code-proxy binary (requires Go 1.26.3+ locally)
make docker         # build Docker image locally (recommended — bundles correct toolchain)
make docker-local   # Dockerfile.local: creds.yaml + certs/ baked in, local use only (never push)
make test           # go test -race with coverage
make lint           # golangci-lint v2
make mocks          # regenerate mockery mocks
make swagger        # regenerate OpenAPI 3.1 spec (swag v2 --v3.1)
make licenses       # regenerate THIRD_PARTY_NOTICES.md
make coverage       # print coverage summary
```

## Quality Gates (Required Before Push)

**MANDATORY**: You MUST run `.claude/hooks/pre-push-checks.sh` (or the equivalent checks below) and confirm all pass before pushing code to the remote. Do not push if any check fails. If your platform has a pre-push hook mechanism, configure it to run this script.

```bash
goimports -l .              # must produce no output
golangci-lint run           # zero issues
gocyclo -over 9 ./internal/ # must produce no output (max complexity: 9)
go test -count=1 ./...      # all tests pass
```

Remediation: `make fmt` auto-fixes formatting (addlicense + goimports + go fmt). `make lint` checks copyright headers + golangci-lint. `make cyclo` checks complexity.

**For AI coding agents**: If your platform supports pre-push hooks or tool-use guards, configure them to run `.claude/hooks/pre-push-checks.sh` before any `git push`. The script exits 0 on success and prints diagnostic output on failure. Claude Code enforcement is pre-configured in `.claude/settings.json`.

## OpenAPI Spec

The API spec lives in `docs/swagger.json` and `docs/swagger.yaml` (OpenAPI 3.1.0). It is generated from annotations on handler functions using [swaggo/swag v2](https://github.com/swaggo/swag). Run `make swagger` after changing any endpoint annotations. CI will fail if the committed spec doesn't match what the code produces.

## Testing Patterns

- No table-driven tests. Inline helpers with `t.Helper()`.
- Integration tests via `httptest.Server` with mock dependencies.
- Mock injection: `MockOAuthService` for handler/redirect tests, `MockHTTPClient` for userinfo tests.
- Coverage target: 80% minimum.
- Always run with `-race`.

## Code Standards

- Go 1.22+ idioms: generics, structured concurrency, error wrapping.
- Structured logging via `zap` with `.Named()` scopes. No `fmt.Sprintf` in log messages.
- JSON tags: lowerCamelCase (`json:"appID"` not `json:"app_id"`).
- Cyclomatic complexity: max 10 per function.
- No `panic` in library code. Wrap errors with context.
- Context is always the first parameter. Never stored in structs.

## Security

- TLS >= 1.2 for all outbound connections.
- No credentials in code or tests — use obviously fake fixture values.
- 1MB max request body enforced via middleware.
- Security headers (X-Content-Type-Options, X-Frame-Options) on all responses.
- `InsecureSkipVerify` is user-configurable but defaults to false.

## CI

GitHub Actions workflow at `.github/workflows/ci.yml` runs:
- Build verification
- Tests with race detection + coverage threshold (80%)
- golangci-lint v2
- OpenAPI spec freshness check

Keep the Makefile and CI in sync — every verification the CI runs should be reproducible locally via `make`.
