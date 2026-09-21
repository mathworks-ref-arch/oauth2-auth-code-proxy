// Copyright (c) 2020-2026, The MathWorks, Inc. All rights reserved.

package app

import (
	"context"
	"fmt"
	"io"
	"slices"

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

	cfg, err := proxy.LoadConfig(args, getenv)
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

	return server.ListenAndServe(ctx, listenConfig(cfg))
}

func handleHelpFlags(args []string, getenv func(string) string, stdout io.Writer) bool {
	if hasArg(args, "--help") || hasArg(args, "-h") || (len(args) == 1 && getenv("OAUTH_CREDS_FILE") == "") {
		proxy.PrintHelp(stdout, getenv("LANG"), Version)
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

func listenConfig(cfg *proxy.Config) domain.ListenConfig {
	return domain.ListenConfig{
		RESTPort:     cfg.Port,
		MWLegacyPort: cfg.MWLegacyPort,
		TLSCert:      cfg.TLSCertFile,
		TLSKey:       cfg.TLSKeyFile,
	}
}

func hasArg(args []string, flag string) bool {
	return slices.Contains(args, flag)
}
