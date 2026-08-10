// Copyright (c) 2020-2026, The MathWorks, Inc. All rights reserved.

package app

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRun_NoArgs_PrintsHelp(t *testing.T) {
	r := require.New(t)
	var buf bytes.Buffer
	err := Run(context.Background(), []string{"cmd"}, func(string) string { return "" }, &buf)
	r.NoError(err)
	r.Contains(buf.String(), "OAuth2 Token Proxy")
	r.Contains(buf.String(), "--config")
}

func TestRun_NoArgs_French(t *testing.T) {
	r := require.New(t)
	var buf bytes.Buffer
	getenv := func(key string) string {
		if key == "LANG" {
			return "fr-FR"
		}
		return ""
	}
	err := Run(context.Background(), []string{"cmd"}, getenv, &buf)
	r.NoError(err)
	r.Contains(buf.String(), "Proxy de jetons OAuth2")
}

func TestRun_MissingConfig(t *testing.T) {
	r := require.New(t)
	var buf bytes.Buffer
	err := Run(context.Background(), []string{"cmd", "--port", "9000"}, func(string) string { return "" }, &buf)
	r.Error(err)
	r.Contains(err.Error(), "credentials file required")
	r.Contains(err.Error(), "--help")
}

func TestRun_InvalidConfigFile(t *testing.T) {
	r := require.New(t)
	var buf bytes.Buffer
	err := Run(context.Background(), []string{"cmd", "--config", "/nonexistent/path.yaml"}, func(string) string { return "" }, &buf)
	r.Error(err)
	r.Contains(err.Error(), "load apps")
}

func TestRun_ValidConfigStartsServer(t *testing.T) {
	r := require.New(t)
	var buf bytes.Buffer

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := Run(ctx, []string{"cmd", "--config", "../testdata/creds.yaml", "--port", "0", "--mwlegacy-port", "0"}, func(string) string { return "" }, &buf)
	r.NoError(err)
}

func TestRun_PortConflict(t *testing.T) {
	r := require.New(t)
	var buf bytes.Buffer

	err := Run(context.Background(), []string{"cmd", "--config", "../testdata/creds.yaml", "--port", "9999", "--mwlegacy-port", "9999"}, func(string) string { return "" }, &buf)
	r.Error(err)
	r.Contains(err.Error(), "cannot be the same")
}

func TestRun_LegacyHelp(t *testing.T) {
	r := require.New(t)
	var buf bytes.Buffer

	err := Run(context.Background(), []string{"cmd", "--legacy"}, func(string) string { return "" }, &buf)
	r.NoError(err)
	r.Contains(buf.String(), "Legacy Settings")
	r.Contains(buf.String(), "oas.port")
	r.Contains(buf.String(), "mito.http.tls")
	r.Contains(buf.String(), "MW_PORT")
}
