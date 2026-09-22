// Copyright (c) 2020-2026, The MathWorks, Inc. All rights reserved.

package proxy

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestLoadConfig_Defaults(t *testing.T) {
	r := require.New(t)

	cfg, err := LoadConfig([]string{"cmd"}, func(string) string { return "" })
	r.NoError(err)
	r.Equal("80", cfg.Port)
	r.Empty(cfg.MWLegacyPort)
	r.False(cfg.MWLegacyEnabled())
}

func TestLoadConfig_ArgsOverride(t *testing.T) {
	r := require.New(t)

	cfg, err := LoadConfig([]string{"cmd", "--port", "9090", "--config", "/tmp/test.yaml"}, func(string) string { return "" })
	r.NoError(err)
	r.Equal("9090", cfg.Port)
	r.Equal("/tmp/test.yaml", cfg.CredSetupFile)
}

func TestLoadConfig_EnvOverride(t *testing.T) {
	r := require.New(t)

	getenv := func(key string) string {
		switch key {
		case "OAUTH_PORT":
			return "8080"
		case "OAUTH_CREDS_FILE":
			return "/env/creds.yaml"
		}
		return ""
	}

	cfg, err := LoadConfig([]string{"cmd"}, getenv)
	r.NoError(err)
	r.Equal("8080", cfg.Port)
	r.Equal("/env/creds.yaml", cfg.CredSetupFile)
}

func TestLoadConfig_MWLegacyPort_EnablesLegacy(t *testing.T) {
	r := require.New(t)

	cfg, err := LoadConfig([]string{"cmd", "--mwlegacy-port", "30045"}, func(string) string { return "" })
	r.NoError(err)
	r.True(cfg.MWLegacyEnabled())
	r.Equal("30045", cfg.MWLegacyPort)
}

func TestLoadConfig_MWLegacyPort_Env(t *testing.T) {
	r := require.New(t)

	getenv := func(key string) string {
		if key == "OAUTH_MWLEGACY_PORT" {
			return "30045"
		}
		return ""
	}

	cfg, err := LoadConfig([]string{"cmd"}, getenv)
	r.NoError(err)
	r.True(cfg.MWLegacyEnabled())
	r.Equal("30045", cfg.MWLegacyPort)
}

func TestLoadConfig_TLSInsecureSkipVerify_Arg(t *testing.T) {
	r := require.New(t)

	cfg, err := LoadConfig([]string{"cmd", "--tls-insecure-skip-verify"}, func(string) string { return "" })
	r.NoError(err)
	r.True(cfg.TLSInsecureSkipVerify)
}

func TestLoadConfig_TLSInsecureSkipVerify_Env(t *testing.T) {
	r := require.New(t)

	getenv := func(key string) string {
		if key == "OAUTH_TLS_INSECURE_SKIP_VERIFY" {
			return "true"
		}
		return ""
	}

	cfg, err := LoadConfig([]string{"cmd"}, getenv)
	r.NoError(err)
	r.True(cfg.TLSInsecureSkipVerify)
}

func TestLoadConfig_TLSInsecureSkipVerify_DefaultFalse(t *testing.T) {
	r := require.New(t)

	cfg, err := LoadConfig([]string{"cmd"}, func(string) string { return "" })
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

func TestLoadConfig_TLSCertAndKey_Args(t *testing.T) {
	r := require.New(t)

	cfg, err := LoadConfig([]string{"cmd", "--tls-cert", "/certs/tls.crt", "--tls-key", "/certs/tls.key"}, func(string) string { return "" })
	r.NoError(err)
	r.Equal("/certs/tls.crt", cfg.TLSCertFile)
	r.Equal("/certs/tls.key", cfg.TLSKeyFile)
}

func TestLoadConfig_TLSCertAndKey_Env(t *testing.T) {
	r := require.New(t)

	getenv := func(key string) string {
		switch key {
		case "OAUTH_TLS_CERT_FILE":
			return "/env/tls.crt"
		case "OAUTH_TLS_KEY_FILE":
			return "/env/tls.key"
		}
		return ""
	}

	cfg, err := LoadConfig([]string{"cmd"}, getenv)
	r.NoError(err)
	r.Equal("/env/tls.crt", cfg.TLSCertFile)
	r.Equal("/env/tls.key", cfg.TLSKeyFile)
}

func TestLoadConfig_TLSCertAndKey_DefaultEmpty(t *testing.T) {
	r := require.New(t)

	cfg, err := LoadConfig([]string{"cmd"}, func(string) string { return "" })
	r.NoError(err)
	r.Empty(cfg.TLSCertFile)
	r.Empty(cfg.TLSKeyFile)
}

func TestLoadConfig_PortResolution(t *testing.T) {
	cases := []struct {
		name         string
		port         string
		mwlegacyPort string
		wantPort     string
		wantMWPort   string
		wantError    string
	}{
		{"defaults_no_mwlegacy", "", "", "80", "", ""},
		{"port_set", "9000", "", "9000", "", ""},
		{"mwport_set", "", "9000", "80", "9000", ""},
		{"both_different", "9000", "8080", "9000", "8080", ""},
		{"both_same", "9000", "9000", "", "", "cannot be the same"},
		{"port_matches_mwport", "30045", "30045", "", "", "cannot be the same"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := require.New(t)

			args := []string{"cmd"}
			if tc.port != "" {
				args = append(args, "--port", tc.port)
			}
			if tc.mwlegacyPort != "" {
				args = append(args, "--mwlegacy-port", tc.mwlegacyPort)
			}

			cfg, err := LoadConfig(args, func(string) string { return "" })

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

func TestLoadConfig_Precedence(t *testing.T) {
	cases := []struct {
		name  string
		args  []string
		env   map[string]string
		check func(*testing.T, *Config)
	}{
		{
			"OAUTH_PORT env overrides CLI --port",
			[]string{"cmd", "--port", "3000"},
			map[string]string{"OAUTH_PORT": "4000"},
			func(t *testing.T, cfg *Config) {
				require.Equal(t, "4000", cfg.Port)
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
			"CLI --tls-cert sets TLSCertFile",
			[]string{"cmd", "--tls-cert", "/new.pem"},
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
			"OAUTH_MWLEGACY_PORT env overrides CLI --mwlegacy-port",
			[]string{"cmd", "--mwlegacy-port", "30045"},
			map[string]string{"OAUTH_MWLEGACY_PORT": "31000"},
			func(t *testing.T, cfg *Config) {
				require.Equal(t, "31000", cfg.MWLegacyPort)
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
			cfg, err := LoadConfig(tc.args, getenv)
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
