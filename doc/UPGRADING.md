# Upgrading from Internal Beta to v1.15+

This guide covers migrating from the internal beta release (which used `MW_*` environment variables, `--set oacp.*` CLI flags, and `app.yaml` config files) to the current release.

## Breaking Change in v1.15

All legacy configuration paths have been **removed**. If you are using any of the following, you **must** migrate before upgrading:

| Removed | Replacement |
|---------|-------------|
| `MW_PORT` env var | `OAUTH_PORT` |
| `MW_CREDSETUPFILE` env var | `OAUTH_CREDS_FILE` |
| `MW_APPFILE` env var | `OAUTH_CREDS_FILE` |
| `--set oacp.port=N` | `--port N` |
| `--set oacp.credSetupFile=path` | `--config path` |
| `--set oacp.appFile=path` | `--config path` directly |
| `app.yaml` config file | Use `--config` and env vars directly |
| `--legacy` help flag | Removed (no legacy options remain) |

## Migration Steps

### 1. Replace environment variables

**Before:**
```yaml
env:
  - name: MW_PORT
    value: "30045"
  - name: MW_CREDSETUPFILE
    value: /etc/oauth2-auth-code-proxy/creds.yaml
```

**After:**
```yaml
env:
  - name: OAUTH_PORT
    value: "30045"
  - name: OAUTH_CREDS_FILE
    value: /etc/oauth2-auth-code-proxy/creds.yaml
```

### 2. Replace CLI flags

**Before:**
```bash
/oauth2-auth-code-proxy --set oacp.port=30045 --set oacp.credSetupFile=/path/to/creds.yaml
```

**After:**
```bash
/oauth2-auth-code-proxy --port 30045 --config /path/to/creds.yaml
```

### 3. Eliminate app.yaml

The internal beta supported a two-file config: `app.yaml` (port + creds path) plus the creds YAML. The current release only needs the creds file — port is set via `--port` or `OAUTH_PORT`.

**Before:**
```yaml
# app.yaml
port: "30045"
credSetupFile: /path/to/creds.yaml
```
```bash
/oauth2-auth-code-proxy --set oacp.appFile=/path/to/app.yaml
```

**After:**
```bash
/oauth2-auth-code-proxy --config /path/to/creds.yaml --port 30045
```

Or equivalently with environment variables:
```bash
OAUTH_CREDS_FILE=/path/to/creds.yaml OAUTH_PORT=30045 /oauth2-auth-code-proxy
```

### 4. Docker run command

**Before:**
```bash
docker run -d \
  -p 30045:30045 \
  -v /path/to/creds.yaml:/etc/oauth2-auth-code-proxy/creds.yaml:ro \
  -e MW_CREDSETUPFILE=/etc/oauth2-auth-code-proxy/creds.yaml \
  oauth2-auth-code-proxy:latest
```

**After:**
```bash
docker run -d \
  -p 30045:30045 \
  -v /path/to/creds.yaml:/etc/oauth2-auth-code-proxy/creds.yaml:ro \
  -e OAUTH_CREDS_FILE=/etc/oauth2-auth-code-proxy/creds.yaml \
  oauth2-auth-code-proxy:latest
```

## Important: `redirectUri` must match your protocol

The `redirectUri` in your `creds.yaml` **must** match how the proxy is actually serving traffic:

- Running **without TLS**: `redirectUri: "http://your-host:30045/oauth/v1/redirect"`
- Running **with TLS**: `redirectUri: "https://your-host:30045/oauth/v1/redirect"`

This URI is sent to the OAuth provider (Google, Microsoft, etc.) and must also be registered as an authorized redirect URI in the provider's console. If these don't match, the OAuth flow will fail with a `redirect_uri_mismatch` error.

## Verifying the Migration

The `/oauth/v1/admin/alive` endpoint confirms the service is healthy:

```bash
curl http://localhost:30045/oauth/v1/admin/alive
```
