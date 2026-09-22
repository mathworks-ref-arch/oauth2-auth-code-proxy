# Testing with MATLAB

How to connect a local MATLAB&reg; desktop client to a running oauth2-auth-code-proxy instance.

## Prerequisites

1. The proxy is running and reachable from your machine (Docker, k8s, or standalone binary).
2. You know the proxy's base URL (e.g., `https://localhost:30045/oauth/v1/api`).
3. The credentials YAML has a valid OAuth app configured (see README.md).
4. MATLAB R2026a or later.

## Quick Test (HTTPS with self-signed cert)

Generate a local TLS certificate (one-time, idempotent):

```bash
make certs
```

This creates `certs/tls.crt` and `certs/tls.key` with SANs for `localhost` and `127.0.0.1`. The cert is gitignored but persists across rebuilds.

Build and run the Docker image (the `docker` target depends on `certs`):

```bash
make docker

docker run -d --name oauth2-proxy-test \
  -p 30045:30045 -p 80:80 \
  -v "$(pwd)/creds.yaml:/etc/oauth2-auth-code-proxy/creds.yaml:ro" \
  -e OAUTH_CREDS_FILE=/etc/oauth2-auth-code-proxy/creds.yaml \
  -e OAUTH_TLS_CERT_FILE=/etc/oauth2-auth-code-proxy/tls.crt \
  -e OAUTH_TLS_KEY_FILE=/etc/oauth2-auth-code-proxy/tls.key \
  oauth2-auth-code-proxy:latest
```

Connect from MATLAB using the local cert file:

```matlab
serverLoc = "https://localhost:30045/oauth/v1/api";
certPath = fullfile(pwd, "certs", "tls.crt");

matlab.internal.feature("GOOGLESHEETS", 1);
matlab.io.internal.googlesheets.oas.activateConnection(serverLoc, 'serverCertificate', certPath);
```

The `'serverCertificate'` parameter tells MATLAB to trust this specific certificate for the HTTPS connection. It is required for self-signed certs.

Once authenticated, you can access Google Sheets directly:

```matlab
>> readtable("https://docs.google.com/spreadsheets/d/1AbC2dEf3GhI4jKlMnOpQrStUvWxYz5678901234567/")

ans =

  3x2 table

    Var1    Var2
    ____    _____

     1      {'A'}
     2      {'B'}
     3      {'C'}
```

## HTTP (No TLS)

If you omit `OAUTH_TLS_CERT_FILE` and `OAUTH_TLS_KEY_FILE`, the service listens on plain HTTP. This works for production behind a load balancer that terminates TLS, or in fully local testing where MATLAB doesn't enforce HTTPS:

```bash
docker run -d --name oauth2-proxy-test \
  -p 30045:30045 \
  -v "$(pwd)/creds.yaml:/etc/oauth2-auth-code-proxy/creds.yaml:ro" \
  -e OAUTH_CREDS_FILE=/etc/oauth2-auth-code-proxy/creds.yaml \
  oauth2-auth-code-proxy:latest
```

```matlab
serverLoc = "http://localhost:30045/oauth/v1/api";

matlab.internal.feature("GOOGLESHEETS", 1);
matlab.io.internal.googlesheets.oas.activateConnection(serverLoc);
```

## Using Your Own Certificate

To use a custom certificate (e.g., signed by your org's CA), mount it and point the env vars at it:

```bash
docker run -d --name oauth2-proxy-test \
  -p 30045:30045 \
  -v "$(pwd)/creds.yaml:/etc/oauth2-auth-code-proxy/creds.yaml:ro" \
  -v "/path/to/my.crt:/certs/server.crt:ro" \
  -v "/path/to/my.key:/certs/server.key:ro" \
  -e OAUTH_CREDS_FILE=/etc/oauth2-auth-code-proxy/creds.yaml \
  -e OAUTH_TLS_CERT_FILE=/certs/server.crt \
  -e OAUTH_TLS_KEY_FILE=/certs/server.key \
  oauth2-auth-code-proxy:latest
```

If the cert is signed by a trusted CA, MATLAB doesn't need the `'serverCertificate'` argument.

## TLS Configuration Summary

| Environment Variable | CLI Flag | Description |
|---------------------|----------|-------------|
| `OAUTH_TLS_CERT_FILE` | `--tls-cert <path>` | Path to TLS certificate for HTTPS serving |
| `OAUTH_TLS_KEY_FILE` | `--tls-key <path>` | Path to TLS private key for HTTPS serving |

Both must be set to enable HTTPS. If either is missing, the service falls back to plain HTTP.

## Troubleshooting

| Symptom | Likely cause |
|---------|--------------|
| Connection refused | Proxy not running, wrong port, or firewall blocking |
| TLS handshake error | Missing `'serverCertificate'` arg in MATLAB, or cert doesn't match the hostname |
| `oauth_services::Unexpected error` | MATLAB&reg; Connector&trade; (MSH) not installed or not connected to the correct MATLAB. Install from https://www.mathworks.com/products/matlab-drive.html |
| Browser opens but MATLAB never gets tokens | Redirect URI mismatch in credentials YAML — `redirectUri` must match what the proxy is actually reachable at |
| "feature not found" error | MATLAB version doesn't support `GOOGLESHEETS` feature flag |

## Verifying the Proxy is Up

Before testing from MATLAB, confirm the service is healthy:

```bash
# HTTP
curl http://localhost:30045/oauth/v1/admin/alive

# HTTPS (self-signed cert)
curl --cacert certs/tls.crt https://localhost:30045/oauth/v1/admin/alive
```

Check that `connCheck` entries show `"success": true` for the Google endpoints.
