// Copyright (c) 2020-2026, The MathWorks, Inc. All rights reserved.

package app

import (
	"context"
	"fmt"
	"io"

	proxy "github.com/mathworks-ref-arch/container-images/oauth2-auth-code-proxy/internal"
	"github.com/mathworks-ref-arch/container-images/oauth2-auth-code-proxy/internal/domain"
	"go.uber.org/zap"
)

var (
	Version    = "dev"
	BuildTime  = "unknown"
	ModuleName = "oauth2-auth-code-proxy"
	GroupID    = ""
)

func Run(ctx context.Context, args []string, getenv func(string) string, stdout io.Writer) error {
	logger, err := zap.NewProduction()
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer logger.Sync() //nolint:errcheck

	cfg, err := proxy.LoadConfig(args, getenv, logger)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if handled := handleHelpFlags(args, getenv, stdout); handled {
		return nil
	}

	if cfg.CredSetupFile == "" {
		return fmt.Errorf("credentials file required: use --config or OAUTH_CREDS_FILE (see --help)")
	}

	apps, err := proxy.LoadApps(cfg.CredSetupFile, logger)
	if err != nil {
		return fmt.Errorf("load apps: %w", err)
	}

	httpCfg := domain.HTTPClientConfig{
		Timeout:               cfg.HTTPClientTimeout,
		ResponseHeaderTimeout: cfg.HTTPResponseHeaderTimeout,
		IdleConnTimeout:       cfg.HTTPIdleConnTimeout,
		TLSHandshakeTimeout:   cfg.HTTPTLSHandshakeTimeout,
		TLSInsecureSkipVerify: cfg.TLSInsecureSkipVerify,
	}

	svc := proxy.NewOAuthService(apps, httpCfg, logger.Named("oauth"))
	server := NewServer(svc, apps, buildInfo(), logger.Named("server"))

	return server.ListenAndServe(ctx, listenConfig(cfg, logger))
}

func handleHelpFlags(args []string, getenv func(string) string, stdout io.Writer) bool {
	if hasArg(args, "--help") || hasArg(args, "-h") || (len(args) == 1 && getenv("OAUTH_CREDS_FILE") == "") {
		proxy.PrintHelp(stdout, getenv("LANG"), Version)
		return true
	}
	if hasArg(args, "--legacy") {
		proxy.PrintHelpLegacy(stdout, getenv("LANG"), Version)
		return true
	}
	return false
}

func buildInfo() domain.BuildInfo {
	return domain.BuildInfo{
		Version:    Version,
		BuildTime:  BuildTime,
		ModuleName: ModuleName,
		GroupID:    GroupID,
	}
}

func listenConfig(cfg *proxy.Config, logger *zap.Logger) domain.ListenConfig {
	tlsCert := cfg.TLSCertFile
	tlsKey := cfg.TLSKeyFile
	if cfg.MWLegacyTLSDisabled {
		tlsCert = ""
		tlsKey = ""
		logger.Warn("TLS disabled via --tls-disabled")
	}
	return domain.ListenConfig{
		RESTPort:        cfg.Port,
		MWLegacyPort:    cfg.MWLegacyPort,
		MWLegacyEnabled: cfg.MWLegacyEnabled,
		TLSCert:         tlsCert,
		TLSKey:          tlsKey,
	}
}

func hasArg(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}
