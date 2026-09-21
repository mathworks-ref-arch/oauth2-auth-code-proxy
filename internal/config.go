// Copyright (c) 2020-2026, The MathWorks, Inc. All rights reserved.

package proxy

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/mathworks-ref-arch/container-images/oauth2-auth-code-proxy/internal/domain"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Port                      string        `json:"port"`
	MWLegacyPort              string        `json:"mwlegacyPort"`
	CredSetupFile         string        `json:"credSetupFile"`
	TLSCertFile           string        `json:"tlsCertFile"`
	TLSKeyFile            string        `json:"tlsKeyFile"`
	TLSInsecureSkipVerify bool          `json:"tlsInsecureSkipVerify"`
	HTTPClientTimeout         time.Duration `json:"httpClientTimeout"`
	HTTPResponseHeaderTimeout time.Duration `json:"httpResponseHeaderTimeout"`
	HTTPIdleConnTimeout       time.Duration `json:"httpIdleConnTimeout"`
	HTTPTLSHandshakeTimeout   time.Duration `json:"httpTLSHandshakeTimeout"`
}

func (c *Config) MWLegacyEnabled() bool {
	return c.MWLegacyPort != ""
}

type AppConfig = domain.AppConfig

type AppInfo = domain.AppInfo

func DefaultConfig() Config {
	return Config{
		Port:                      "80",
		HTTPClientTimeout:         30 * time.Second,
		HTTPResponseHeaderTimeout: 10 * time.Second,
		HTTPIdleConnTimeout:       90 * time.Second,
		HTTPTLSHandshakeTimeout:   10 * time.Second,
	}
}

func LoadConfig(args []string, getenv func(string) string) (*Config, error) {
	cfg := DefaultConfig()

	loadConfigFromArgs(&cfg, args)
	loadConfigFromEnv(&cfg, getenv)

	if err := validatePorts(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func validatePorts(cfg *Config) error {
	if cfg.MWLegacyEnabled() && cfg.Port == cfg.MWLegacyPort && cfg.Port != "0" {
		return fmt.Errorf("port and mwlegacy-port cannot be the same")
	}
	return nil
}

func loadConfigFromArgs(cfg *Config, args []string) {
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--tls-insecure-skip-verify":
			cfg.TLSInsecureSkipVerify = true
		default:
			if v, ok := consumeValueArg(args, &i); ok {
				applyValueArg(cfg, args[i-1], v)
			}
		}
	}
}

func consumeValueArg(args []string, i *int) (string, bool) {
	arg := args[*i]
	if *i+1 >= len(args) {
		return "", false
	}
	switch {
	case strings.HasPrefix(arg, "--port") && !strings.HasPrefix(arg, "--port-"):
	case strings.HasPrefix(arg, "--mwlegacy-port"):
	case strings.HasPrefix(arg, "--config"):
	case strings.HasPrefix(arg, "--tls-cert"):
	case strings.HasPrefix(arg, "--tls-key"):
	default:
		return "", false
	}
	*i++
	return args[*i], true
}

func applyValueArg(cfg *Config, flag string, val string) {
	switch {
	case strings.HasPrefix(flag, "--port"):
		cfg.Port = val
	case strings.HasPrefix(flag, "--mwlegacy-port"):
		cfg.MWLegacyPort = val
	case strings.HasPrefix(flag, "--config"):
		cfg.CredSetupFile = val
	case strings.HasPrefix(flag, "--tls-cert"):
		cfg.TLSCertFile = val
	case strings.HasPrefix(flag, "--tls-key"):
		cfg.TLSKeyFile = val
	}
}

func loadConfigFromEnv(cfg *Config, getenv func(string) string) {
	if v := getenv("OAUTH_PORT"); v != "" {
		cfg.Port = v
	}
	if v := getenv("OAUTH_MWLEGACY_PORT"); v != "" {
		cfg.MWLegacyPort = v
	}
	if v := getenv("OAUTH_CREDS_FILE"); v != "" {
		cfg.CredSetupFile = v
	}
	if v := getenv("OAUTH_TLS_CERT_FILE"); v != "" {
		cfg.TLSCertFile = v
	}
	if v := getenv("OAUTH_TLS_KEY_FILE"); v != "" {
		cfg.TLSKeyFile = v
	}
	if v := getenv("OAUTH_TLS_INSECURE_SKIP_VERIFY"); v == "1" || v == "true" {
		cfg.TLSInsecureSkipVerify = true
	}
}

var readFile = os.ReadFile

func LoadApps(filePath string, logger *zap.Logger) (map[string]AppConfig, error) {
	apps := map[string]AppConfig{}

	b, err := readFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read creds file: %w", err)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(b))
	for {
		var app AppConfig
		if err := decoder.Decode(&app); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("decode creds yaml: %w", err)
		}
		if app.ID != "" {
			apps[app.ID] = app
		}
	}

	if err := validateApps(apps); err != nil {
		return nil, err
	}

	return apps, nil
}

func validateApps(apps map[string]AppConfig) error {
	if len(apps) != 1 {
		return fmt.Errorf("exactly one app must be defined in creds.yaml, got %d", len(apps))
	}
	if _, ok := apps["100"]; !ok {
		return fmt.Errorf("app with id '100' must be defined in creds.yaml")
	}

	for _, app := range apps {
		if strings.Contains(app.ID, "-") {
			return fmt.Errorf("app id cannot contain hyphens: %s", app.ID)
		}

		if err := validateAppFields(app); err != nil {
			return err
		}

		if !slices.Contains(app.Scopes, "email") {
			return fmt.Errorf("app %s: 'email' scope is required", app.ID)
		}
	}

	return nil
}

func validateAppFields(app AppConfig) error {
	val := reflect.ValueOf(app)
	typ := reflect.TypeFor[AppConfig]()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldName := typ.Field(i).Name

		if field.Kind() == reflect.String {
			if strings.TrimSpace(field.String()) == "" {
				return fmt.Errorf("app %s: field %s is required", app.ID, fieldName)
			}
		} else if field.Kind() == reflect.Slice && field.Type().Elem().Kind() == reflect.String {
			if field.Len() == 0 {
				return fmt.Errorf("app %s: field %s must not be empty", app.ID, fieldName)
			}
			for j := 0; j < field.Len(); j++ {
				if strings.TrimSpace(field.Index(j).String()) == "" {
					return fmt.Errorf("app %s: field %s contains empty string", app.ID, fieldName)
				}
			}
		}
	}
	return nil
}

var AppConfigToInfo = domain.AppConfigToInfo
