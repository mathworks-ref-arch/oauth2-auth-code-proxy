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

	optConfig      string
	optPort        string
	optMWLegacy    string
	optTLSCert     string
	optTLSKey      string
	optTLSInsecure string

	envCredsFile   string
	envPort        string
	envMWLegacy    string
	envTLSCert     string
	envTLSKey      string
	envTLSInsecure string
	envLang        string

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

	optConfig:      "  --config <path>              Credentials YAML file (required)",
	optPort:        "  --port <port>                REST API listen port (default: 80)",
	optMWLegacy:    "  --mwlegacy-port <port>       Also serve POST /oauth/v1/api on this port",
	optTLSCert:     "  --tls-cert <path>            TLS certificate for HTTPS serving",
	optTLSKey:      "  --tls-key <path>             TLS private key for HTTPS serving",
	optTLSInsecure: "  --tls-insecure-skip-verify   Skip TLS verify on outbound calls",

	envCredsFile:   "  OAUTH_CREDS_FILE             Credentials YAML file path",
	envPort:        "  OAUTH_PORT                   REST API port (default: 80)",
	envMWLegacy:    "  OAUTH_MWLEGACY_PORT          Also serve POST /oauth/v1/api on this port",
	envTLSCert:     "  OAUTH_TLS_CERT_FILE          TLS certificate path",
	envTLSKey:      "  OAUTH_TLS_KEY_FILE           TLS private key path",
	envTLSInsecure: "  OAUTH_TLS_INSECURE_SKIP_VERIFY  Set true to skip outbound TLS verify",
	envLang:        "  LANG                         Locale for help output (en-US, fr-FR)",

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

	optConfig:      "  --config <chemin>            Fichier YAML des identifiants (requis)",
	optPort:        "  --port <port>                Port d'écoute API REST (défaut : 80)",
	optMWLegacy:    "  --mwlegacy-port <port>       Servir aussi POST /oauth/v1/api sur ce port",
	optTLSCert:     "  --tls-cert <chemin>          Certificat TLS pour le service HTTPS",
	optTLSKey:      "  --tls-key <chemin>           Clé privée TLS pour le service HTTPS",
	optTLSInsecure: "  --tls-insecure-skip-verify   Ignorer la vérification TLS sur les appels sortants",

	envCredsFile:   "  OAUTH_CREDS_FILE             Chemin du fichier YAML des identifiants",
	envPort:        "  OAUTH_PORT                   Port API REST (défaut : 80)",
	envMWLegacy:    "  OAUTH_MWLEGACY_PORT          Servir aussi POST /oauth/v1/api sur ce port",
	envTLSCert:     "  OAUTH_TLS_CERT_FILE          Chemin du certificat TLS",
	envTLSKey:      "  OAUTH_TLS_KEY_FILE           Chemin de la clé privée TLS",
	envTLSInsecure: "  OAUTH_TLS_INSECURE_SKIP_VERIFY  Mettre à true pour ignorer la vérification TLS sortante",
	envLang:        "  LANG                         Langue de l'aide (en-US, fr-FR)",

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

//nolint:errcheck
func PrintHelp(w io.Writer, lang string, ver string) {
	s := getStrings(lang)

	fmt.Fprintf(w, s.title+"\n\n", ver)
	fmt.Fprintf(w, "%s\n\n", s.description)

	fmt.Fprintf(w, "%s\n", s.usage)
	fmt.Fprintf(w, "  oauth2-auth-code-proxy --config <creds.yaml> [options]\n\n")

	fmt.Fprintf(w, "%s\n", s.options)
	fmt.Fprintf(w, "%s\n", s.optConfig)
	fmt.Fprintf(w, "%s\n", s.optPort)
	fmt.Fprintf(w, "%s\n", s.optMWLegacy)
	fmt.Fprintf(w, "%s\n", s.optTLSCert)
	fmt.Fprintf(w, "%s\n", s.optTLSKey)
	fmt.Fprintf(w, "%s\n\n", s.optTLSInsecure)

	fmt.Fprintf(w, "%s\n", s.envVars)
	fmt.Fprintf(w, "%s\n", s.envCredsFile)
	fmt.Fprintf(w, "%s\n", s.envPort)
	fmt.Fprintf(w, "%s\n", s.envMWLegacy)
	fmt.Fprintf(w, "%s\n", s.envTLSCert)
	fmt.Fprintf(w, "%s\n", s.envTLSKey)
	fmt.Fprintf(w, "%s\n", s.envTLSInsecure)
	fmt.Fprintf(w, "%s\n\n", s.envLang)

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
