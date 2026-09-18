// Copyright (c) 2020-2026, The MathWorks, Inc. All rights reserved.

package proxy

import (
	"fmt"
	"io"
	"strings"
)

type helpStrings struct {
	title       string
	description string
	usage       string
	options     string
	envVars     string
	examples    string
	langNote    string
	copyright   string
	license     string

	optConfig          string
	optPort            string
	optMwlegacyPort    string
	optMwlegacyEnabled string
	optTlsCert         string
	optTlsKey          string
	optTlsDisabled     string
	optTlsInsecure     string
	optLegacy          string

	envCredsFile       string
	envPort            string
	envMwlegacyPort    string
	envMwlegacyEnabled string
	envTlsCert         string
	envTlsKey          string
	envTlsDisabled     string
	envTlsInsecure     string
	envLang            string

	legacyDefault  string
	legacyDisable  string
	legacyTitle    string
	legacyPrefixed string
	legacyBackward string
	legacyOasPort  string
	legacyOasCred  string
	legacyOasApp   string
	legacyMitoTls  string
	legacyEnvTitle string
	legacyMwPort   string
	legacyMwCred   string
	legacyMwApp    string

	exDocker    string
	exDockerRun string
	exDockerVol string
	exDockerEnv string
	exDockerImg string
	exBinary    string
	exBinaryCmd string
}

var defaultStrings = helpStrings{
	title:       "OAuth2 Token Proxy (v%s)",
	description: "OAuth2 token exchange proxy for desktop clients. Handles the server-side\nOAuth2 authorization code flow with PKCE so desktop applications can\nauthenticate with third-party cloud providers without embedding client secrets.",
	usage:       "Usage:",
	options:     "Options:",
	envVars:     "Environment Variables:",
	examples:    "Examples:",
	langNote:    "Set LANG environment variable for localized output (e.g., LANG=fr-FR).",
	copyright:   "Copyright (c) 2020-2026, The MathWorks, Inc. All rights reserved.",
	license:     "Governed by the MathWorks Cloud Reference Architecture License.",

	optConfig:          "  --config <path>              Credentials YAML file (required)",
	optPort:            "  --port <port>                REST API listen port (default: 80)",
	optMwlegacyPort:    "  --mwlegacy-port <port>       Legacy API listen port (default: 30045)",
	optMwlegacyEnabled: "  --mwlegacy-enabled=false     Disable legacy API listener",
	optTlsCert:         "  --tls-cert <path>            TLS certificate for HTTPS serving",
	optTlsKey:          "  --tls-key <path>             TLS private key for HTTPS serving",
	optTlsDisabled:     "  --tls-disabled               Disable TLS even if certs present",
	optTlsInsecure:     "  --tls-insecure-skip-verify   Skip TLS verify on outbound calls",
	optLegacy:          "  --legacy                     Show legacy configuration options",

	envCredsFile:       "  OAUTH_CREDS_FILE             Credentials YAML file path",
	envPort:            "  OAUTH_PORT                   REST API port (default: 80)",
	envMwlegacyPort:    "  OAUTH_MWLEGACY_PORT          Legacy API port (default: 30045)",
	envMwlegacyEnabled: "  OAUTH_MWLEGACY_ENABLED       Set false to disable legacy API",
	envTlsCert:         "  OAUTH_TLS_CERT_FILE          TLS certificate path",
	envTlsKey:          "  OAUTH_TLS_KEY_FILE           TLS private key path",
	envTlsDisabled:     "  OAUTH_TLS_DISABLED           Set true to disable TLS even if certs present",
	envTlsInsecure:     "  OAUTH_TLS_INSECURE_SKIP_VERIFY  Set true to skip outbound TLS verify",
	envLang:            "  LANG                         Locale for help output (en-US, fr-FR)",

	legacyDefault:  "The legacy API listener is enabled by default on port 30045.",
	legacyDisable:  "Use --mwlegacy-enabled=false to disable, or --legacy for legacy options.",
	legacyTitle:    "Legacy Settings (--set):",
	legacyPrefixed: "  Settings prefixed with oas. or mito. are deprecated but supported",
	legacyBackward: "  for backward compatibility.",
	legacyOasPort:  "  --set oas.port=<port>                       Legacy API port (use --mwlegacy-port)",
	legacyOasCred:  "  --set oas.credSetupFile=<path>               Credentials file (use --config)",
	legacyOasApp:   "  --set oas.appFile=<path>                     App config file (deprecated)",
	legacyMitoTls:  "  --set mito.http.tls.enabled=false            Disable TLS (use --tls-disabled)",
	legacyEnvTitle: "Legacy Environment Variables:",
	legacyMwPort:   "  MW_PORT                      Legacy API port (use OAUTH_MWLEGACY_PORT)",
	legacyMwCred:   "  MW_CREDSETUPFILE             Credentials file (use OAUTH_CREDS_FILE)",
	legacyMwApp:    "  MW_APPFILE                   App config file (deprecated)",

	exDocker:    "  # Docker with mounted credentials",
	exDockerRun: "  docker run -p 80:80 -p 30045:30045 \\",
	exDockerVol: "    -v ./creds.yaml:/etc/oauth2-auth-code-proxy/creds.yaml:ro \\",
	exDockerEnv: "    -e OAUTH_CREDS_FILE=/etc/oauth2-auth-code-proxy/creds.yaml \\",
	exDockerImg: "    oauth2-auth-code-proxy:latest",
	exBinary:    "  # Binary with HTTPS",
	exBinaryCmd: "  oauth2-auth-code-proxy --config creds.yaml --tls-cert tls.crt --tls-key tls.key",
}

var frenchStrings = helpStrings{
	title:       "Proxy de jetons OAuth2 (v%s)",
	description: "Proxy d'échange de jetons OAuth2 pour les clients de bureau. Gère le flux\nd'autorisation OAuth2 côté serveur avec PKCE afin que les applications de\nbureau puissent s'authentifier auprès de fournisseurs cloud tiers sans\nintégrer de secrets client.",
	usage:       "Utilisation :",
	options:     "Options :",
	envVars:     "Variables d'environnement :",
	examples:    "Exemples :",
	langNote:    "Définissez la variable d'environnement LANG pour une sortie localisée (ex : LANG=fr-FR).",
	copyright:   "Copyright (c) 2020-2026, The MathWorks, Inc. Tous droits réservés.",
	license:     "Régi par la licence MathWorks Cloud Reference Architecture.",

	optConfig:          "  --config <chemin>            Fichier YAML des identifiants (requis)",
	optPort:            "  --port <port>                Port d'écoute API REST (défaut : 80)",
	optMwlegacyPort:    "  --mwlegacy-port <port>       Port d'écoute API legacy (défaut : 30045)",
	optMwlegacyEnabled: "  --mwlegacy-enabled=false     Désactiver l'écouteur API legacy",
	optTlsCert:         "  --tls-cert <chemin>          Certificat TLS pour le service HTTPS",
	optTlsKey:          "  --tls-key <chemin>           Clé privée TLS pour le service HTTPS",
	optTlsDisabled:     "  --tls-disabled               Désactiver TLS même si certs présents",
	optTlsInsecure:     "  --tls-insecure-skip-verify   Ignorer la vérification TLS sur les appels sortants",
	optLegacy:          "  --legacy                     Afficher les options de configuration legacy",

	envCredsFile:       "  OAUTH_CREDS_FILE             Chemin du fichier YAML des identifiants",
	envPort:            "  OAUTH_PORT                   Port API REST (défaut : 80)",
	envMwlegacyPort:    "  OAUTH_MWLEGACY_PORT          Port API legacy (défaut : 30045)",
	envMwlegacyEnabled: "  OAUTH_MWLEGACY_ENABLED       Mettre à false pour désactiver l'API legacy",
	envTlsCert:         "  OAUTH_TLS_CERT_FILE          Chemin du certificat TLS",
	envTlsKey:          "  OAUTH_TLS_KEY_FILE           Chemin de la clé privée TLS",
	envTlsDisabled:     "  OAUTH_TLS_DISABLED           Mettre à true pour désactiver TLS même si certs présents",
	envTlsInsecure:     "  OAUTH_TLS_INSECURE_SKIP_VERIFY  Mettre à true pour ignorer la vérification TLS sortante",
	envLang:            "  LANG                         Langue de l'aide (en-US, fr-FR)",

	legacyDefault:  "L'écouteur API legacy est activé par défaut sur le port 30045.",
	legacyDisable:  "Utilisez --mwlegacy-enabled=false pour désactiver, ou --legacy pour les options legacy.",
	legacyTitle:    "Paramètres legacy (--set) :",
	legacyPrefixed: "  Les paramètres préfixés par oas. ou mito. sont obsolètes mais supportés",
	legacyBackward: "  pour la rétrocompatibilité.",
	legacyOasPort:  "  --set oas.port=<port>                       Port API legacy (utiliser --mwlegacy-port)",
	legacyOasCred:  "  --set oas.credSetupFile=<chemin>             Fichier identifiants (utiliser --config)",
	legacyOasApp:   "  --set oas.appFile=<chemin>                   Fichier config app (obsolète)",
	legacyMitoTls:  "  --set mito.http.tls.enabled=false            Désactiver TLS (utiliser --tls-disabled)",
	legacyEnvTitle: "Variables d'environnement legacy :",
	legacyMwPort:   "  MW_PORT                      Port API legacy (utiliser OAUTH_MWLEGACY_PORT)",
	legacyMwCred:   "  MW_CREDSETUPFILE             Fichier identifiants (utiliser OAUTH_CREDS_FILE)",
	legacyMwApp:    "  MW_APPFILE                   Fichier config app (obsolète)",

	exDocker:    "  # Docker avec identifiants montés",
	exDockerRun: "  docker run -p 80:80 -p 30045:30045 \\",
	exDockerVol: "    -v ./creds.yaml:/etc/oauth2-auth-code-proxy/creds.yaml:ro \\",
	exDockerEnv: "    -e OAUTH_CREDS_FILE=/etc/oauth2-auth-code-proxy/creds.yaml \\",
	exDockerImg: "    oauth2-auth-code-proxy:latest",
	exBinary:    "  # Binaire avec HTTPS",
	exBinaryCmd: "  oauth2-auth-code-proxy --config creds.yaml --tls-cert tls.crt --tls-key tls.key",
}

func getStrings(lang string) *helpStrings {
	if strings.HasPrefix(strings.ToLower(lang), "fr") {
		return &frenchStrings
	}
	return &defaultStrings
}

func PrintHelp(w io.Writer, lang string, ver string) {
	printHelp(w, lang, ver, false)
}

func PrintHelpLegacy(w io.Writer, lang string, ver string) {
	printHelp(w, lang, ver, true)
}

//nolint:errcheck
func printHelp(w io.Writer, lang string, ver string, legacy bool) {
	s := getStrings(lang)

	fmt.Fprintf(w, s.title+"\n\n", ver)
	fmt.Fprintf(w, "%s\n\n", s.description)

	fmt.Fprintf(w, "%s\n", s.usage)
	fmt.Fprintf(w, "  oauth2-auth-code-proxy --config <creds.yaml> [options]\n\n")

	fmt.Fprintf(w, "%s\n", s.options)
	fmt.Fprintf(w, "%s\n", s.optConfig)
	fmt.Fprintf(w, "%s\n", s.optPort)
	fmt.Fprintf(w, "%s\n", s.optMwlegacyPort)
	fmt.Fprintf(w, "%s\n", s.optMwlegacyEnabled)
	fmt.Fprintf(w, "%s\n", s.optTlsCert)
	fmt.Fprintf(w, "%s\n", s.optTlsKey)
	fmt.Fprintf(w, "%s\n", s.optTlsDisabled)
	fmt.Fprintf(w, "%s\n", s.optTlsInsecure)
	fmt.Fprintf(w, "%s\n\n", s.optLegacy)

	fmt.Fprintf(w, "%s\n", s.envVars)
	fmt.Fprintf(w, "%s\n", s.envCredsFile)
	fmt.Fprintf(w, "%s\n", s.envPort)
	fmt.Fprintf(w, "%s\n", s.envMwlegacyPort)
	fmt.Fprintf(w, "%s\n", s.envMwlegacyEnabled)
	fmt.Fprintf(w, "%s\n", s.envTlsCert)
	fmt.Fprintf(w, "%s\n", s.envTlsKey)
	fmt.Fprintf(w, "%s\n", s.envTlsDisabled)
	fmt.Fprintf(w, "%s\n", s.envTlsInsecure)
	fmt.Fprintf(w, "%s\n\n", s.envLang)

	if !legacy {
		fmt.Fprintf(w, "%s\n", s.legacyDefault)
		fmt.Fprintf(w, "%s\n\n", s.legacyDisable)
	}

	if legacy {
		fmt.Fprintf(w, "%s\n", s.legacyTitle)
		fmt.Fprintf(w, "%s\n", s.legacyPrefixed)
		fmt.Fprintf(w, "%s\n\n", s.legacyBackward)
		fmt.Fprintf(w, "%s\n", s.legacyOasPort)
		fmt.Fprintf(w, "%s\n", s.legacyOasCred)
		fmt.Fprintf(w, "%s\n", s.legacyOasApp)
		fmt.Fprintf(w, "%s\n\n", s.legacyMitoTls)
		fmt.Fprintf(w, "%s\n", s.legacyEnvTitle)
		fmt.Fprintf(w, "%s\n", s.legacyMwPort)
		fmt.Fprintf(w, "%s\n", s.legacyMwCred)
		fmt.Fprintf(w, "%s\n\n", s.legacyMwApp)
	}

	fmt.Fprintf(w, "%s\n", s.examples)
	fmt.Fprintf(w, "%s\n", s.exDocker)
	fmt.Fprintf(w, "%s\n", s.exDockerRun)
	fmt.Fprintf(w, "%s\n", s.exDockerVol)
	fmt.Fprintf(w, "%s\n", s.exDockerEnv)
	fmt.Fprintf(w, "%s\n\n", s.exDockerImg)
	fmt.Fprintf(w, "%s\n", s.exBinary)
	fmt.Fprintf(w, "%s\n\n", s.exBinaryCmd)

	fmt.Fprintf(w, "%s\n\n", s.langNote)

	fmt.Fprintf(w, "%s\n", s.copyright)
	fmt.Fprintf(w, "%s\n", s.license)
	fmt.Fprintf(w, "See https://github.com/mathworks-ref-arch/container-images/oauth2-auth-code-proxy/blob/main/LICENSE.md\n")
}
