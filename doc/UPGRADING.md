# Upgrading from Internal Beta to v1.15

This guide covers migrating from the internal beta release (which used `MW_*` environment variables, `--set oacp.*` CLI flags, and `app.yaml` config files) to the current open-source release.

## Summary of Changes

| Internal Beta | v1.15 (OSS) | Notes |
|---------------|-------------|-------|
| `MW_PORT` | `OAUTH_PORT` | Environment variable |
| `MW_CREDSETUPFILE` | `OAUTH_CREDS_FILE` | Environment variable |
| `MW_APPFILE` + app.yaml | `--config` / `OAUTH_CREDS_FILE` | App.yaml no longer needed |
| `--set oacp.port=N` | `--port N` | CLI flag |
| `--set oacp.credSetupFile=path` | `--config path` | CLI flag |
| `--set oacp.appFile=path` | _(removed)_ | Use `--config` directly |

## What Still Works (Deprecated)

The internal beta configuration paths still function in v1.15 but emit deprecation warnings at startup. They will be removed in a future release.

If you see messages like these in your logs, follow this guide to migrate:

```
WARN  deprecated: MW_PORT is deprecated, use OAUTH_PORT instead
WARN  deprecated: MW_CREDSETUPFILE is deprecated, use OAUTH_CREDS_FILE instead
WARN  deprecated: --set oacp.port is deprecated, use --port instead
WARN  deprecated: app.yaml config file is deprecated, use --config and env vars directly
```

## Migration Steps

### 1. Replace environment variables

**Before (internal beta):**
```yaml
env:
  - name: MW_PORT
    value: "30045"
  - name: MW_CREDSETUPFILE
    value: /etc/oauth2-auth-code-proxy/creds.yaml
```

**After (v1.15):**
```yaml
env:
  - name: OAUTH_PORT
    value: "30045"
  - name: OAUTH_CREDS_FILE
    value: /etc/oauth2-auth-code-proxy/creds.yaml
```

### 2. Replace CLI flags

**Before (internal beta):**
```bash
/oauth2-auth-code-proxy --set oacp.port=30045 --set oacp.credSetupFile=/path/to/creds.yaml
```

**After (v1.15):**
```bash
/oauth2-auth-code-proxy --port 30045 --config /path/to/creds.yaml
```

### 3. Eliminate app.yaml

The internal beta supported a two-file config: `app.yaml` (port + creds path) plus the creds YAML. The OSS release only needs the creds file — port is set via `--port` or `OAUTH_PORT`.

**Before (internal beta):**
```yaml
# app.yaml
port: "30045"
credSetupFile: /path/to/creds.yaml
```
```bash
/oauth2-auth-code-proxy --set oacp.appFile=/path/to/app.yaml
```

**After (v1.15):**
```bash
/oauth2-auth-code-proxy --config /path/to/creds.yaml --port 30045
```

Or equivalently with environment variables:
```bash
OAUTH_CREDS_FILE=/path/to/creds.yaml OAUTH_PORT=30045 /oauth2-auth-code-proxy
```

### 4. Docker run command

**Before (internal beta):**
```bash
docker run -d \
  -p 30045:30045 \
  -v /path/to/creds.yaml:/etc/oauth2-auth-code-proxy/creds.yaml:ro \
  -e MW_CREDSETUPFILE=/etc/oauth2-auth-code-proxy/creds.yaml \
  oauth2-auth-code-proxy:latest
```

**After (v1.15):**
```bash
docker run -d \
  -p 30045:30045 \
  -v /path/to/creds.yaml:/etc/oauth2-auth-code-proxy/creds.yaml:ro \
  -e OAUTH_CREDS_FILE=/etc/oauth2-auth-code-proxy/creds.yaml \
  oauth2-auth-code-proxy:latest
```

### 5. TLS skip-verify (if used)

**Before:** No equivalent existed in the internal beta.

**After (v1.15):**
```bash
--tls-insecure-skip-verify
# or
OAUTH_TLS_INSECURE_SKIP_VERIFY=true
```

### 6. Helm values

If deploying with the Helm chart, the values file uses the OSS-style configuration directly:

```yaml
config:
  port: "30045"
  tlsInsecureSkipVerify: false

credentials:
  existingSecret: my-oauth-creds
```

No `MW_*` environment variables are needed in the Helm deployment.

## Precedence (if mixing old and new)

During migration, you may have both old and new config sources active. Later sources override earlier ones:

1. Built-in defaults (port=30045)
2. `--set oacp.*` CLI flags (deprecated)
3. `MW_*` environment variables (deprecated)
4. App YAML file (deprecated)
5. `--port` / `--config` CLI flags
6. `OAUTH_*` environment variables (highest priority)

This means you can add the new-style config alongside old-style and the new values will take precedence, allowing a gradual rollout.

## Important: `redirectUri` must match your protocol

The `redirectUri` in your `creds.yaml` **must** match how the proxy is actually serving traffic:

- Running **without TLS**: `redirectUri: "http://your-host:30045/oauth/v1/redirect"`
- Running **with TLS**: `redirectUri: "https://your-host:30045/oauth/v1/redirect"`

This URI is sent to the OAuth provider (Google, Microsoft, etc.) and must also be registered as an authorized redirect URI in the provider's console. If these don't match, the OAuth flow will fail with a `redirect_uri_mismatch` error.

## Verifying the Migration

After updating your configuration, confirm the service starts without deprecation warnings:

```bash
docker logs <container-name> 2>&1 | grep -i deprecated
```

If no output, the migration is complete. The `/oauth/v1/admin/alive` endpoint confirms the service is healthy:

```bash
curl http://localhost:30045/oauth/v1/admin/alive
```
