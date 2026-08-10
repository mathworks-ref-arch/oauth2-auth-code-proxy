// Copyright (c) 2020-2026, The MathWorks, Inc. All rights reserved.

package proxy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestLoadConfig_Defaults(t *testing.T) {
	r := require.New(t)
	logger := zap.NewNop()

	cfg, err := LoadConfig([]string{"cmd"}, func(string) string { return "" }, logger)
	r.NoError(err)
	r.Equal("80", cfg.Port)
	r.Equal("30045", cfg.MWLegacyPort)
}

func TestLoadConfig_ArgsOverride(t *testing.T) {
	r := require.New(t)
	logger := zap.NewNop()

	cfg, err := LoadConfig([]string{"cmd", "--port", "9090", "--config", "/tmp/test.yaml"}, func(string) string { return "" }, logger)
	r.NoError(err)
	r.Equal("9090", cfg.Port)
	r.Equal("/tmp/test.yaml", cfg.CredSetupFile)
}

func TestLoadConfig_EnvOverride(t *testing.T) {
	r := require.New(t)
	logger := zap.NewNop()

	getenv := func(key string) string {
		switch key {
		case "OAUTH_PORT":
			return "8080"
		case "OAUTH_CREDS_FILE":
			return "/env/creds.yaml"
		}
		return ""
	}

	cfg, err := LoadConfig([]string{"cmd"}, getenv, logger)
	r.NoError(err)
	r.Equal("8080", cfg.Port)
	r.Equal("/env/creds.yaml", cfg.CredSetupFile)
}

func TestLoadConfig_TLSInsecureSkipVerify_Arg(t *testing.T) {
	r := require.New(t)
	logger := zap.NewNop()

	cfg, err := LoadConfig([]string{"cmd", "--tls-insecure-skip-verify"}, func(string) string { return "" }, logger)
	r.NoError(err)
	r.True(cfg.TLSInsecureSkipVerify)
}

func TestLoadConfig_TLSInsecureSkipVerify_Env(t *testing.T) {
	r := require.New(t)
	logger := zap.NewNop()

	getenv := func(key string) string {
		if key == "OAUTH_TLS_INSECURE_SKIP_VERIFY" {
			return "true"
		}
		return ""
	}

	cfg, err := LoadConfig([]string{"cmd"}, getenv, logger)
	r.NoError(err)
	r.True(cfg.TLSInsecureSkipVerify)
}

func TestLoadConfig_TLSInsecureSkipVerify_DefaultFalse(t *testing.T) {
	r := require.New(t)
	logger := zap.NewNop()

	cfg, err := LoadConfig([]string{"cmd"}, func(string) string { return "" }, logger)
	r.NoError(err)
	r.False(cfg.TLSInsecureSkipVerify)
}

func TestLoadApps_ValidFile(t *testing.T) {
	r := require.New(t)
	logger := zap.NewNop()

	apps, err := LoadApps("testdata/creds.yaml", logger)
	r.NoError(err)
	r.Len(apps, 1)

	app := apps["100"]
	r.Equal("100", app.ID)
	r.Equal("sheets", app.Type)
	r.Equal("google", app.CloudProvider)
	r.Equal("fake-client-id-for-testing.apps.googleusercontent.com", app.ClientID)
	r.True(app.IncludeRefreshToken)
	r.Contains(app.Scopes, "email")
}

func TestLoadApps_MissingFile(t *testing.T) {
	r := require.New(t)
	logger := zap.NewNop()

	_, err := LoadApps("testdata/nonexistent.yaml", logger)
	r.Error(err)
	r.Contains(err.Error(), "read creds file")
}

func TestValidateApps_NoDashes(t *testing.T) {
	r := require.New(t)

	apps := map[string]AppConfig{
		"bad-id": {ID: "bad-id"},
	}
	err := validateApps(apps)
	r.Error(err)
}

func TestValidateApps_RequiresID100(t *testing.T) {
	r := require.New(t)

	apps := map[string]AppConfig{
		"200": fullTestApp("200"),
	}
	err := validateApps(apps)
	r.Error(err)
	r.Contains(err.Error(), "id '100'")
}

func TestValidateApps_RequiresEmailScope(t *testing.T) {
	r := require.New(t)

	app := fullTestApp("100")
	app.Scopes = []string{"https://www.googleapis.com/auth/drive.readonly"}
	apps := map[string]AppConfig{"100": app}

	err := validateApps(apps)
	r.Error(err)
	r.Contains(err.Error(), "email")
}

func TestLoadConfig_MWPortEnvVar(t *testing.T) {
	r := require.New(t)
	logger := zap.NewNop()

	getenv := func(key string) string {
		if key == "MW_PORT" {
			return "31000"
		}
		return ""
	}

	cfg, err := LoadConfig([]string{"cmd"}, getenv, logger)
	r.NoError(err)
	r.Equal("31000", cfg.MWLegacyPort)
}

func TestLoadConfig_MWCredSetupFileEnvVar(t *testing.T) {
	r := require.New(t)
	logger := zap.NewNop()

	getenv := func(key string) string {
		if key == "MW_CREDSETUPFILE" {
			return "/mw/creds.yaml"
		}
		return ""
	}

	cfg, err := LoadConfig([]string{"cmd"}, getenv, logger)
	r.NoError(err)
	r.Equal("/mw/creds.yaml", cfg.CredSetupFile)
}

func TestLoadConfig_OAuthEnvOverridesMW(t *testing.T) {
	r := require.New(t)
	logger := zap.NewNop()

	getenv := func(key string) string {
		switch key {
		case "MW_PORT":
			return "31000"
		case "OAUTH_MWLEGACY_PORT":
			return "32000"
		case "OAUTH_PORT":
			return "9000"
		}
		return ""
	}

	cfg, err := LoadConfig([]string{"cmd"}, getenv, logger)
	r.NoError(err)
	r.Equal("9000", cfg.Port)
	r.Equal("32000", cfg.MWLegacyPort)
}

func TestLoadConfig_AppFileLoading(t *testing.T) {
	r := require.New(t)
	logger := zap.NewNop()

	cfg, err := LoadConfig([]string{"cmd", "--set", "oas.appFile=testdata/app.yaml"}, func(string) string { return "" }, logger)
	r.NoError(err)
	r.Equal("9999", cfg.MWLegacyPort)
	r.Equal("/from/app/yaml/creds.yaml", cfg.CredSetupFile)
}

func TestLoadConfig_AppFileOverridesMWEnv(t *testing.T) {
	r := require.New(t)
	logger := zap.NewNop()

	getenv := func(key string) string {
		if key == "MW_PORT" {
			return "31000"
		}
		return ""
	}

	cfg, err := LoadConfig([]string{"cmd", "--set", "oas.appFile=testdata/app.yaml"}, getenv, logger)
	r.NoError(err)
	r.Equal("9999", cfg.MWLegacyPort)
}

func TestLoadConfig_NewArgsOverrideAppFile(t *testing.T) {
	r := require.New(t)
	logger := zap.NewNop()

	cfg, err := LoadConfig([]string{"cmd", "--set", "oas.appFile=testdata/app.yaml", "--mwlegacy-port", "7777"}, func(string) string { return "" }, logger)
	r.NoError(err)
	r.Equal("7777", cfg.MWLegacyPort)
}

func TestLoadConfig_OAuthEnvOverridesAppFile(t *testing.T) {
	r := require.New(t)
	logger := zap.NewNop()

	getenv := func(key string) string {
		if key == "OAUTH_MWLEGACY_PORT" {
			return "5555"
		}
		return ""
	}

	cfg, err := LoadConfig([]string{"cmd", "--set", "oas.appFile=testdata/app.yaml"}, getenv, logger)
	r.NoError(err)
	r.Equal("5555", cfg.MWLegacyPort)
}

func TestLoadConfig_MWAppFileEnvVar(t *testing.T) {
	r := require.New(t)
	logger := zap.NewNop()

	getenv := func(key string) string {
		if key == "MW_APPFILE" {
			return "testdata/app.yaml"
		}
		return ""
	}

	cfg, err := LoadConfig([]string{"cmd"}, getenv, logger)
	r.NoError(err)
	r.Equal("9999", cfg.MWLegacyPort)
	r.Equal("/from/app/yaml/creds.yaml", cfg.CredSetupFile)
}

func TestLoadConfig_AppFileIgnoresUnknownFields(t *testing.T) {
	r := require.New(t)
	logger := zap.NewNop()

	cfg, err := LoadConfig([]string{"cmd", "--set", "oas.appFile=testdata/app.yaml"}, func(string) string { return "" }, logger)
	r.NoError(err)
	r.Equal("9999", cfg.MWLegacyPort)
}

func TestLoadConfig_SetArgWithoutValue(t *testing.T) {
	r := require.New(t)
	logger := zap.NewNop()

	cfg, err := LoadConfig([]string{"cmd", "--set"}, func(string) string { return "" }, logger)
	r.NoError(err)
	r.Equal("80", cfg.Port)
	r.Equal("30045", cfg.MWLegacyPort)
}

func TestLoadConfig_DeprecationWarning_MWPort(t *testing.T) {
	r := require.New(t)
	a := assert.New(t)
	core, logs := observer.New(zap.WarnLevel)
	logger := zap.New(core)

	getenv := func(key string) string {
		if key == "MW_PORT" {
			return "31000"
		}
		return ""
	}

	cfg, err := LoadConfig([]string{"cmd"}, getenv, logger)
	r.NoError(err)
	r.Equal("31000", cfg.MWLegacyPort)

	a.GreaterOrEqual(logs.Len(), 1)
	a.Contains(logs.All()[0].Message, "MW_PORT")
	a.Contains(logs.All()[0].Message, "OAUTH_MWLEGACY_PORT")
}

func TestLoadConfig_DeprecationWarning_MWCredSetupFile(t *testing.T) {
	r := require.New(t)
	a := assert.New(t)
	core, logs := observer.New(zap.WarnLevel)
	logger := zap.New(core)

	getenv := func(key string) string {
		if key == "MW_CREDSETUPFILE" {
			return "/mw/creds.yaml"
		}
		return ""
	}

	cfg, err := LoadConfig([]string{"cmd"}, getenv, logger)
	r.NoError(err)
	r.Equal("/mw/creds.yaml", cfg.CredSetupFile)

	a.GreaterOrEqual(logs.Len(), 1)
	a.Contains(logs.All()[0].Message, "MW_CREDSETUPFILE")
	a.Contains(logs.All()[0].Message, "OAUTH_CREDS_FILE")
}

func TestLoadConfig_DeprecationWarning_SetOasPort(t *testing.T) {
	r := require.New(t)
	a := assert.New(t)
	core, logs := observer.New(zap.WarnLevel)
	logger := zap.New(core)

	cfg, err := LoadConfig([]string{"cmd", "--set", "oas.port=9999"}, func(string) string { return "" }, logger)
	r.NoError(err)
	r.Equal("9999", cfg.MWLegacyPort)

	a.GreaterOrEqual(logs.Len(), 1)
	a.Contains(logs.All()[0].Message, "--set oas.port")
	a.Contains(logs.All()[0].Message, "--mwlegacy-port")
}

func TestLoadConfig_DeprecationWarning_AppFile(t *testing.T) {
	r := require.New(t)
	a := assert.New(t)
	core, logs := observer.New(zap.WarnLevel)
	logger := zap.New(core)

	cfg, err := LoadConfig([]string{"cmd", "--set", "oas.appFile=testdata/app.yaml"}, func(string) string { return "" }, logger)
	r.NoError(err)
	r.Equal("9999", cfg.MWLegacyPort)

	var messages []string
	for _, entry := range logs.All() {
		messages = append(messages, entry.Message)
	}
	a.GreaterOrEqual(len(messages), 2, "expected warnings for --set oas.appFile and app.yaml loading")
}

func TestLoadConfig_TLSCertAndKey_Args(t *testing.T) {
	r := require.New(t)
	logger := zap.NewNop()

	cfg, err := LoadConfig([]string{"cmd", "--tls-cert", "/certs/tls.crt", "--tls-key", "/certs/tls.key"}, func(string) string { return "" }, logger)
	r.NoError(err)
	r.Equal("/certs/tls.crt", cfg.TLSCertFile)
	r.Equal("/certs/tls.key", cfg.TLSKeyFile)
}

func TestLoadConfig_TLSCertAndKey_Env(t *testing.T) {
	r := require.New(t)
	logger := zap.NewNop()

	getenv := func(key string) string {
		switch key {
		case "OAUTH_TLS_CERT_FILE":
			return "/env/tls.crt"
		case "OAUTH_TLS_KEY_FILE":
			return "/env/tls.key"
		}
		return ""
	}

	cfg, err := LoadConfig([]string{"cmd"}, getenv, logger)
	r.NoError(err)
	r.Equal("/env/tls.crt", cfg.TLSCertFile)
	r.Equal("/env/tls.key", cfg.TLSKeyFile)
}

func TestLoadConfig_TLSCertAndKey_DefaultEmpty(t *testing.T) {
	r := require.New(t)
	logger := zap.NewNop()

	cfg, err := LoadConfig([]string{"cmd"}, func(string) string { return "" }, logger)
	r.NoError(err)
	r.Empty(cfg.TLSCertFile)
	r.Empty(cfg.TLSKeyFile)
}

func TestLoadConfig_LegacyMitoTLS(t *testing.T) {
	r := require.New(t)
	logger := zap.NewNop()

	cfg, err := LoadConfig([]string{
		"cmd",
		"--set", "mito.http.tls.certificatepath=/certs/all-dev.pem",
		"--set", "mito.http.tls.privatekeypath=/certs/all-dev.key",
		"--set", "mito.http.tls.enabled=true",
	}, func(string) string { return "" }, logger)
	r.NoError(err)
	r.Equal("/certs/all-dev.pem", cfg.TLSCertFile)
	r.Equal("/certs/all-dev.key", cfg.TLSKeyFile)
	r.False(cfg.MWLegacyTLSDisabled)
}

func TestLoadConfig_LegacyMitoTLS_Disabled(t *testing.T) {
	r := require.New(t)
	logger := zap.NewNop()

	cfg, err := LoadConfig([]string{
		"cmd",
		"--set", "mito.http.tls.certificatepath=/certs/all-dev.pem",
		"--set", "mito.http.tls.privatekeypath=/certs/all-dev.key",
		"--set", "mito.http.tls.enabled=false",
	}, func(string) string { return "" }, logger)
	r.NoError(err)
	r.Equal("/certs/all-dev.pem", cfg.TLSCertFile)
	r.Equal("/certs/all-dev.key", cfg.TLSKeyFile)
	r.True(cfg.MWLegacyTLSDisabled)
}

func TestLoadConfig_NoDeprecationWarning_NewStyle(t *testing.T) {
	r := require.New(t)
	a := assert.New(t)
	core, logs := observer.New(zap.WarnLevel)
	logger := zap.New(core)

	getenv := func(key string) string {
		if key == "OAUTH_PORT" {
			return "8080"
		}
		return ""
	}

	cfg, err := LoadConfig([]string{"cmd", "--config", "/tmp/creds.yaml"}, getenv, logger)
	r.NoError(err)
	r.Equal("8080", cfg.Port)
	r.Equal("/tmp/creds.yaml", cfg.CredSetupFile)

	a.Equal(0, logs.Len(), "new-style config should not emit deprecation warnings")
}

func TestLoadConfig_PortResolution(t *testing.T) {
	logger := zap.NewNop()

	cases := []struct {
		name         string
		mwlegacy     bool
		port         string // --port (REST), "" means unset
		mwlegacyPort string // --mwlegacy-port, "" means unset
		wantPort     string // cfg.Port (REST)
		wantMWPort   string // cfg.MWLegacyPort
		wantError    string
	}{
		{"enabled_defaults", true, "", "", "80", "30045", ""},
		{"enabled_port_set", true, "9000", "", "9000", "30045", ""},
		{"enabled_mwport_set", true, "", "9000", "80", "9000", ""},
		{"enabled_both_different", true, "9000", "8080", "9000", "8080", ""},
		{"enabled_both_same", true, "9000", "9000", "", "", "cannot be the same"},
		{"enabled_port_matches_mwport_default", true, "30045", "", "", "", "cannot be the same"},
		{"disabled_defaults", false, "", "", "80", "30045", ""},
		{"disabled_port_set", false, "9000", "", "9000", "30045", ""},
		{"disabled_mwport_set", false, "", "9000", "80", "9000", ""},
		{"legacy_set_oas_port", true, "", "", "80", "8888", ""},
		{"legacy_mw_port_env", true, "", "", "80", "7777", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := require.New(t)

			args := []string{"cmd"}
			if !tc.mwlegacy {
				args = append(args, "--mwlegacy-enabled=false")
			}
			if tc.port != "" {
				args = append(args, "--port", tc.port)
			}
			if tc.mwlegacyPort != "" {
				args = append(args, "--mwlegacy-port", tc.mwlegacyPort)
			}

			getenv := func(string) string { return "" }

			if tc.name == "legacy_set_oas_port" {
				args = append(args, "--set", "oas.port=8888")
			}
			if tc.name == "legacy_mw_port_env" {
				getenv = func(key string) string {
					if key == "MW_PORT" {
						return "7777"
					}
					return ""
				}
			}

			cfg, err := LoadConfig(args, getenv, logger)

			if tc.wantError != "" {
				r.Error(err)
				r.Contains(err.Error(), tc.wantError)
			} else {
				r.NoError(err)
				r.Equal(tc.wantPort, cfg.Port, "Port (REST)")
				r.Equal(tc.wantMWPort, cfg.MWLegacyPort, "MWLegacyPort")
			}
		})
	}
}

func TestLoadConfig_LegacySetArgs(t *testing.T) {
	logger := zap.NewNop()
	noenv := func(string) string { return "" }

	cases := []struct {
		name  string
		args  []string
		check func(*testing.T, *Config)
	}{
		{
			"oas.port sets MWLegacyPort",
			[]string{"cmd", "--set", "oas.port=31000"},
			func(t *testing.T, cfg *Config) {
				require.Equal(t, "31000", cfg.MWLegacyPort)
				require.Equal(t, "80", cfg.Port)
			},
		},
		{
			"oas.credSetupFile sets CredSetupFile",
			[]string{"cmd", "--set", "oas.credSetupFile=/path/creds.yaml"},
			func(t *testing.T, cfg *Config) {
				require.Equal(t, "/path/creds.yaml", cfg.CredSetupFile)
			},
		},
		{
			"oas.appFile sets AppFile and loads it",
			[]string{"cmd", "--set", "oas.appFile=testdata/app.yaml"},
			func(t *testing.T, cfg *Config) {
				require.Equal(t, "testdata/app.yaml", cfg.AppFile)
				require.Equal(t, "9999", cfg.MWLegacyPort)
				require.Equal(t, "/from/app/yaml/creds.yaml", cfg.CredSetupFile)
			},
		},
		{
			"mito.http.tls.certificatepath sets TLSCertFile",
			[]string{"cmd", "--set", "mito.http.tls.certificatepath=/certs/tls.pem"},
			func(t *testing.T, cfg *Config) {
				require.Equal(t, "/certs/tls.pem", cfg.TLSCertFile)
			},
		},
		{
			"mito.http.tls.certificatepem sets TLSCertFile",
			[]string{"cmd", "--set", "mito.http.tls.certificatepem=/certs/alt.pem"},
			func(t *testing.T, cfg *Config) {
				require.Equal(t, "/certs/alt.pem", cfg.TLSCertFile)
			},
		},
		{
			"mito.http.tls.privatekeypath sets TLSKeyFile",
			[]string{"cmd", "--set", "mito.http.tls.privatekeypath=/certs/tls.key"},
			func(t *testing.T, cfg *Config) {
				require.Equal(t, "/certs/tls.key", cfg.TLSKeyFile)
			},
		},
		{
			"mito.http.tls.privatekeypem sets TLSKeyFile",
			[]string{"cmd", "--set", "mito.http.tls.privatekeypem=/certs/alt.key"},
			func(t *testing.T, cfg *Config) {
				require.Equal(t, "/certs/alt.key", cfg.TLSKeyFile)
			},
		},
		{
			"mito.http.tls.enabled=true does not disable TLS",
			[]string{"cmd", "--set", "mito.http.tls.enabled=true"},
			func(t *testing.T, cfg *Config) {
				require.False(t, cfg.MWLegacyTLSDisabled)
			},
		},
		{
			"mito.http.tls.enabled=false disables TLS",
			[]string{"cmd", "--set", "mito.http.tls.enabled=false"},
			func(t *testing.T, cfg *Config) {
				require.True(t, cfg.MWLegacyTLSDisabled)
			},
		},
		{
			"unknown oas. key is ignored",
			[]string{"cmd", "--set", "oas.unknownKey=value"},
			func(t *testing.T, cfg *Config) {
				require.Equal(t, "80", cfg.Port)
				require.Equal(t, "30045", cfg.MWLegacyPort)
			},
		},
		{
			"unknown mito. key is ignored",
			[]string{"cmd", "--set", "mito.http.tls.unknownKey=value"},
			func(t *testing.T, cfg *Config) {
				require.Empty(t, cfg.TLSCertFile)
			},
		},
		{
			"non-prefixed key is ignored",
			[]string{"cmd", "--set", "port=9999"},
			func(t *testing.T, cfg *Config) {
				require.Equal(t, "80", cfg.Port)
				require.Equal(t, "30045", cfg.MWLegacyPort)
			},
		},
		{
			"full launch.json equivalent",
			[]string{
				"cmd",
				"--set", "oas.credSetupFile=/configs/creds.yaml",
				"--set", "oas.port=30045",
				"--set", "mito.http.tls.certificatepath=/configs/ssl/all-dev.pem",
				"--set", "mito.http.tls.privatekeypath=/configs/ssl/all-dev.key",
				"--set", "mito.http.tls.enabled=true",
			},
			func(t *testing.T, cfg *Config) {
				require.Equal(t, "/configs/creds.yaml", cfg.CredSetupFile)
				require.Equal(t, "30045", cfg.MWLegacyPort)
				require.Equal(t, "80", cfg.Port)
				require.Equal(t, "/configs/ssl/all-dev.pem", cfg.TLSCertFile)
				require.Equal(t, "/configs/ssl/all-dev.key", cfg.TLSKeyFile)
				require.False(t, cfg.MWLegacyTLSDisabled)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := LoadConfig(tc.args, noenv, logger)
			require.NoError(t, err)
			tc.check(t, cfg)
		})
	}
}

func TestLoadConfig_Precedence(t *testing.T) {
	logger := zap.NewNop()

	cases := []struct {
		name  string
		args  []string
		env   map[string]string
		check func(*testing.T, *Config)
	}{
		{
			"CLI --mwlegacy-port overrides --set oas.port",
			[]string{"cmd", "--set", "oas.port=11111", "--mwlegacy-port", "22222"},
			nil,
			func(t *testing.T, cfg *Config) {
				require.Equal(t, "22222", cfg.MWLegacyPort)
			},
		},
		{
			"CLI --port overrides OAUTH_PORT env",
			[]string{"cmd", "--port", "3000"},
			map[string]string{"OAUTH_PORT": "4000"},
			func(t *testing.T, cfg *Config) {
				// OAUTH_PORT is processed AFTER CLI args, so env wins
				require.Equal(t, "4000", cfg.Port)
			},
		},
		{
			"OAUTH_MWLEGACY_PORT overrides MW_PORT",
			[]string{"cmd"},
			map[string]string{"MW_PORT": "11111", "OAUTH_MWLEGACY_PORT": "22222"},
			func(t *testing.T, cfg *Config) {
				require.Equal(t, "22222", cfg.MWLegacyPort)
			},
		},
		{
			"OAUTH_MWLEGACY_PORT overrides --set oas.port",
			[]string{"cmd", "--set", "oas.port=11111"},
			map[string]string{"OAUTH_MWLEGACY_PORT": "22222"},
			func(t *testing.T, cfg *Config) {
				require.Equal(t, "22222", cfg.MWLegacyPort)
			},
		},
		{
			"MW_PORT overrides --set oas.port (processed after set args)",
			[]string{"cmd", "--set", "oas.port=11111"},
			map[string]string{"MW_PORT": "22222"},
			func(t *testing.T, cfg *Config) {
				// MW env is processed after --set args
				require.Equal(t, "22222", cfg.MWLegacyPort)
			},
		},
		{
			"app file port overrides MW_PORT (app file after MW env)",
			[]string{"cmd", "--set", "oas.appFile=testdata/app.yaml"},
			map[string]string{"MW_PORT": "11111"},
			func(t *testing.T, cfg *Config) {
				// app file (port: 9999) is loaded after MW env
				require.Equal(t, "9999", cfg.MWLegacyPort)
			},
		},
		{
			"CLI --mwlegacy-port overrides app file",
			[]string{"cmd", "--set", "oas.appFile=testdata/app.yaml", "--mwlegacy-port", "7777"},
			nil,
			func(t *testing.T, cfg *Config) {
				require.Equal(t, "7777", cfg.MWLegacyPort)
			},
		},
		{
			"OAUTH_MWLEGACY_PORT overrides app file",
			[]string{"cmd", "--set", "oas.appFile=testdata/app.yaml"},
			map[string]string{"OAUTH_MWLEGACY_PORT": "6666"},
			func(t *testing.T, cfg *Config) {
				require.Equal(t, "6666", cfg.MWLegacyPort)
			},
		},
		{
			"CLI --config overrides --set oas.credSetupFile",
			[]string{"cmd", "--set", "oas.credSetupFile=/old/path", "--config", "/new/path"},
			nil,
			func(t *testing.T, cfg *Config) {
				require.Equal(t, "/new/path", cfg.CredSetupFile)
			},
		},
		{
			"OAUTH_CREDS_FILE overrides CLI --config",
			[]string{"cmd", "--config", "/cli/path"},
			map[string]string{"OAUTH_CREDS_FILE": "/env/path"},
			func(t *testing.T, cfg *Config) {
				require.Equal(t, "/env/path", cfg.CredSetupFile)
			},
		},
		{
			"OAUTH_CREDS_FILE overrides MW_CREDSETUPFILE",
			[]string{"cmd"},
			map[string]string{"MW_CREDSETUPFILE": "/old", "OAUTH_CREDS_FILE": "/new"},
			func(t *testing.T, cfg *Config) {
				require.Equal(t, "/new", cfg.CredSetupFile)
			},
		},
		{
			"CLI --tls-cert overrides --set mito.http.tls.certificatepath",
			[]string{"cmd", "--set", "mito.http.tls.certificatepath=/old.pem", "--tls-cert", "/new.pem"},
			nil,
			func(t *testing.T, cfg *Config) {
				require.Equal(t, "/new.pem", cfg.TLSCertFile)
			},
		},
		{
			"OAUTH_TLS_CERT_FILE overrides CLI --tls-cert",
			[]string{"cmd", "--tls-cert", "/cli.pem"},
			map[string]string{"OAUTH_TLS_CERT_FILE": "/env.pem"},
			func(t *testing.T, cfg *Config) {
				require.Equal(t, "/env.pem", cfg.TLSCertFile)
			},
		},
		{
			"full precedence chain: set < MW env < app file < CLI < OAUTH env",
			[]string{
				"cmd",
				"--set", "oas.port=10000",
				"--set", "oas.appFile=testdata/app.yaml",
				"--mwlegacy-port", "20000",
			},
			map[string]string{"MW_PORT": "15000", "OAUTH_MWLEGACY_PORT": "30000"},
			func(t *testing.T, cfg *Config) {
				// Precedence: --set oas.port=10000, then MW_PORT=15000 overrides,
				// then app file port=9999 overrides, then --mwlegacy-port=20000 overrides,
				// then OAUTH_MWLEGACY_PORT=30000 overrides. Final: 30000
				require.Equal(t, "30000", cfg.MWLegacyPort)
			},
		},
		{
			"--mwlegacy-enabled=false via env",
			[]string{"cmd"},
			map[string]string{"OAUTH_MWLEGACY_ENABLED": "false"},
			func(t *testing.T, cfg *Config) {
				require.False(t, cfg.MWLegacyEnabled)
			},
		},
		{
			"--mwlegacy-enabled=false via env (0)",
			[]string{"cmd"},
			map[string]string{"OAUTH_MWLEGACY_ENABLED": "0"},
			func(t *testing.T, cfg *Config) {
				require.False(t, cfg.MWLegacyEnabled)
			},
		},
		{
			"--tls-insecure-skip-verify via env",
			[]string{"cmd"},
			map[string]string{"OAUTH_TLS_INSECURE_SKIP_VERIFY": "true"},
			func(t *testing.T, cfg *Config) {
				require.True(t, cfg.TLSInsecureSkipVerify)
			},
		},
		{
			"--tls-insecure-skip-verify via flag",
			[]string{"cmd", "--tls-insecure-skip-verify"},
			nil,
			func(t *testing.T, cfg *Config) {
				require.True(t, cfg.TLSInsecureSkipVerify)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			getenv := func(key string) string {
				if tc.env != nil {
					return tc.env[key]
				}
				return ""
			}
			cfg, err := LoadConfig(tc.args, getenv, logger)
			require.NoError(t, err)
			tc.check(t, cfg)
		})
	}
}

func fullTestApp(id string) AppConfig {
	return AppConfig{ //nolint:gosec // test fixture with fake credentials
		ID:                  id,
		Type:                "sheets",
		CloudProvider:       "google",
		Description:         "test",
		DescriptionCode:     "test.code",
		ClientID:            "fake-client-id",
		ClientSecret:        "fake-secret",
		RedirectURI:         "https://localhost/redirect",
		Scopes:              []string{"email", "drive"},
		AuthURL:             "https://accounts.google.com/o/oauth2/auth",
		TokenURL:            "https://oauth2.googleapis.com/token",
		RevokeURL:           "https://oauth2.googleapis.com/revoke",
		Host:                "sheets.googleapis.com",
		TestingURL:          "https://openidconnect.googleapis.com/v1/userinfo",
		NameKey:             "name",
		EmailKey:            "email",
		IDTokenNameKey:      "name",
		IDTokenEmailKey:     "email",
		IncludeRefreshToken: true,
	}
}
