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
	MWLegacyEnabled           bool          `json:"mwlegacyEnabled"`
	CredSetupFile             string        `json:"credSetupFile"`
	TLSCertFile               string        `json:"tlsCertFile"`
	TLSKeyFile                string        `json:"tlsKeyFile"`
	MWLegacyTLSDisabled       bool          `json:"mwlegacyTlsDisabled"`
	TLSInsecureSkipVerify     bool          `json:"tlsInsecureSkipVerify"`
	HTTPClientTimeout         time.Duration `json:"httpClientTimeout"`
	HTTPResponseHeaderTimeout time.Duration `json:"httpResponseHeaderTimeout"`
	HTTPIdleConnTimeout       time.Duration `json:"httpIdleConnTimeout"`
	HTTPTLSHandshakeTimeout   time.Duration `json:"httpTLSHandshakeTimeout"`
	AppFile                   string        `json:"-"`
}

type AppConfig = domain.AppConfig

type AppInfo = domain.AppInfo

func DefaultConfig() Config {
	return Config{
		Port:                      "80",
		MWLegacyPort:              "30045",
		MWLegacyEnabled:           true,
		HTTPClientTimeout:         30 * time.Second,
		HTTPResponseHeaderTimeout: 10 * time.Second,
		HTTPIdleConnTimeout:       90 * time.Second,
		HTTPTLSHandshakeTimeout:   10 * time.Second,
	}
}

func LoadConfig(args []string, getenv func(string) string, logger *zap.Logger) (*Config, error) {
	cfg := DefaultConfig()

	loadConfigFromSetArgs(&cfg, args, logger)
	loadConfigFromMWEnv(&cfg, getenv, logger)

	if err := loadConfigFromAppFile(&cfg, getenv, logger); err != nil {
		return nil, err
	}

	loadConfigFromNewArgs(&cfg, args)
	loadConfigFromEnv(&cfg, getenv)

	if err := validatePorts(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func validatePorts(cfg *Config) error {
	if cfg.MWLegacyEnabled && cfg.Port == cfg.MWLegacyPort && cfg.Port != "0" {
		return fmt.Errorf("port and mwlegacy-port cannot be the same when mwlegacy is enabled")
	}
	return nil
}

func loadConfigFromSetArgs(cfg *Config, args []string, logger *zap.Logger) {
	for i := 1; i < len(args); i++ {
		if strings.HasPrefix(args[i], "--set") && i+1 < len(args) {
			i++
			parseSetArg(cfg, args[i])
			logSetArgDeprecation(logger, args[i])
		}
	}
}

func logSetArgDeprecation(logger *zap.Logger, arg string) {
	parts := strings.SplitN(arg, "=", 2)

	if key, found := strings.CutPrefix(parts[0], "oas."); found {
		switch key {
		case "port":
			logger.Warn("deprecated: --set oas.port is deprecated, use --mwlegacy-port instead")
		case "credSetupFile":
			logger.Warn("deprecated: --set oas.credSetupFile is deprecated, use --config instead")
		case "appFile":
			logger.Warn("deprecated: --set oas.appFile is deprecated, use --config and env vars directly")
		}
		return
	}

	if strings.HasPrefix(parts[0], "mito.http.tls.") {
		logger.Warn("deprecated: --set mito.http.tls.* is deprecated, use --tls-cert and --tls-key instead",
			zap.String("key", parts[0]))
	}
}

func loadConfigFromNewArgs(cfg *Config, args []string) {
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--tls-insecure-skip-verify":
			cfg.TLSInsecureSkipVerify = true
		case "--mwlegacy-enabled=false":
			cfg.MWLegacyEnabled = false
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
	case strings.HasPrefix(arg, "--set"):
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

func loadConfigFromMWEnv(cfg *Config, getenv func(string) string, logger *zap.Logger) {
	if v := getenv("MW_PORT"); v != "" {
		cfg.MWLegacyPort = v
		logger.Warn("deprecated: MW_PORT is deprecated, use OAUTH_MWLEGACY_PORT instead")
	}
	if v := getenv("MW_CREDSETUPFILE"); v != "" {
		cfg.CredSetupFile = v
		logger.Warn("deprecated: MW_CREDSETUPFILE is deprecated, use OAUTH_CREDS_FILE instead")
	}
	if v := getenv("MW_APPFILE"); v != "" {
		cfg.AppFile = v
		logger.Warn("deprecated: MW_APPFILE is deprecated, use --config and env vars directly")
	}
}

type appFileConfig struct {
	Port          string `yaml:"port"`
	CredSetupFile string `yaml:"credSetupFile"`
}

func loadConfigFromAppFile(cfg *Config, getenv func(string) string, logger *zap.Logger) error {
	if cfg.AppFile == "" {
		if v := getenv("MW_APPFILE"); v != "" {
			cfg.AppFile = v
		}
	}
	if cfg.AppFile == "" {
		return nil
	}

	data, err := readFile(cfg.AppFile)
	if err != nil {
		return fmt.Errorf("read app file %s: %w", cfg.AppFile, err)
	}

	var fc appFileConfig
	if err := yaml.Unmarshal(data, &fc); err != nil {
		return fmt.Errorf("parse app file %s: %w", cfg.AppFile, err)
	}

	if fc.Port != "" {
		cfg.MWLegacyPort = fc.Port
	}
	if fc.CredSetupFile != "" {
		cfg.CredSetupFile = fc.CredSetupFile
	}

	logger.Warn("deprecated: app.yaml config file is deprecated, use --config and env vars directly",
		zap.String("path", cfg.AppFile))
	return nil
}

func parseSetArg(cfg *Config, arg string) {
	parts := strings.SplitN(arg, "=", 2)
	val := ""
	if len(parts) == 2 {
		val = parts[1]
	}

	if key, found := strings.CutPrefix(parts[0], "oas."); found {
		parseOASSetArg(cfg, key, val)
		return
	}

	if key, found := strings.CutPrefix(parts[0], "mito.http.tls."); found {
		parseMitoTLSSetArg(cfg, key, val)
		return
	}
}

func parseOASSetArg(cfg *Config, key string, val string) {
	switch key {
	case "port":
		cfg.MWLegacyPort = val
	case "credSetupFile":
		cfg.CredSetupFile = val
	case "appFile":
		cfg.AppFile = val
	}
}

func parseMitoTLSSetArg(cfg *Config, key string, val string) {
	switch key {
	case "certificatepath", "certificatepem":
		cfg.TLSCertFile = val
	case "privatekeypath", "privatekeypem":
		cfg.TLSKeyFile = val
	case "enabled":
		if val == "false" {
			cfg.MWLegacyTLSDisabled = true
		}
	}
}

func loadConfigFromEnv(cfg *Config, getenv func(string) string) {
	loadStringEnvVars(cfg, getenv)
	if v := getenv("OAUTH_MWLEGACY_ENABLED"); v == "0" || v == "false" {
		cfg.MWLegacyEnabled = false
	}
	if v := getenv("OAUTH_TLS_INSECURE_SKIP_VERIFY"); v == "1" || v == "true" {
		cfg.TLSInsecureSkipVerify = true
	}
}

func loadStringEnvVars(cfg *Config, getenv func(string) string) {
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
