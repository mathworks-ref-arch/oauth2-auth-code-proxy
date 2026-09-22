# Copyright 2024-2026 The MathWorks, Inc.
FROM golang:1.26 AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG BUILD_TIMESTAMP=unknown
ARG GROUP_ID=""

RUN PKG=$(go list -m)/internal/app && CGO_ENABLED=0 go build \
    -ldflags "-X '${PKG}.Version=${VERSION}' -X '${PKG}.BuildTime=${BUILD_TIMESTAMP}' -X '${PKG}.ModuleName=$(go list -m)' -X '${PKG}.GroupID=${GROUP_ID}'" \
    -o /oauth2-auth-code-proxy .

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /oauth2-auth-code-proxy /oauth2-auth-code-proxy
COPY certs/tls.crt /etc/oauth2-auth-code-proxy/tls.crt
COPY certs/tls.key /etc/oauth2-auth-code-proxy/tls.key

EXPOSE 80 30045

ENTRYPOINT ["/oauth2-auth-code-proxy"]
