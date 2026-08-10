// Copyright (c) 2020-2026, The MathWorks, Inc. All rights reserved.

package proxy

import (
	"io"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func PrintHelp(w io.Writer, lang string, ver string) {
	printHelp(w, lang, ver, false)
}

func PrintHelpLegacy(w io.Writer, lang string, ver string) {
	printHelp(w, lang, ver, true)
}

//nolint:errcheck
func printHelp(w io.Writer, lang string, ver string, legacy bool) {
	tag, _ := language.Parse(lang)
	p := message.NewPrinter(tag)

	p.Fprintf(w, "OAuth2 Token Proxy (v%s)\n\n", ver)
	p.Fprintf(w, "OAuth2 token exchange proxy for desktop clients. Handles the server-side\nOAuth2 authorization code flow with PKCE so desktop applications can\nauthenticate with third-party cloud providers without embedding client secrets.\n\n")

	p.Fprintf(w, "Usage:\n")
	p.Fprintf(w, "  oauth2-auth-code-proxy --config <creds.yaml> [options]\n\n")

	p.Fprintf(w, "Options:\n")
	p.Fprintf(w, "  --config <path>              Credentials YAML file (required)\n")
	p.Fprintf(w, "  --port <port>                REST API listen port (default: 80)\n")
	p.Fprintf(w, "  --mwlegacy-port <port>       Legacy API listen port (default: 30045)\n")
	p.Fprintf(w, "  --mwlegacy-enabled=false     Disable legacy API listener\n")
	p.Fprintf(w, "  --tls-cert <path>            TLS certificate for HTTPS serving\n")
	p.Fprintf(w, "  --tls-key <path>             TLS private key for HTTPS serving\n")
	p.Fprintf(w, "  --tls-insecure-skip-verify   Skip TLS verify on outbound calls\n")
	p.Fprintf(w, "  --legacy                     Show legacy configuration options\n\n")

	p.Fprintf(w, "Environment Variables:\n")
	p.Fprintf(w, "  OAUTH_CREDS_FILE             Credentials YAML file path\n")
	p.Fprintf(w, "  OAUTH_PORT                   REST API port (default: 80)\n")
	p.Fprintf(w, "  OAUTH_MWLEGACY_PORT          Legacy API port (default: 30045)\n")
	p.Fprintf(w, "  OAUTH_MWLEGACY_ENABLED       Set false to disable legacy API\n")
	p.Fprintf(w, "  OAUTH_TLS_CERT_FILE          TLS certificate path\n")
	p.Fprintf(w, "  OAUTH_TLS_KEY_FILE           TLS private key path\n")
	p.Fprintf(w, "  OAUTH_TLS_INSECURE_SKIP_VERIFY  Set true to skip outbound TLS verify\n")
	p.Fprintf(w, "  LANG                         Locale for help output (en-US, fr-FR)\n\n")

	if !legacy {
		p.Fprintf(w, "The legacy API listener is enabled by default on port 30045.\n")
		p.Fprintf(w, "Use --mwlegacy-enabled=false to disable, or --legacy for legacy options.\n\n")
	}

	if legacy {
		p.Fprintf(w, "Legacy Settings (--set):\n")
		p.Fprintf(w, "  Settings prefixed with oas. or mito. are deprecated but supported\n")
		p.Fprintf(w, "  for backward compatibility.\n\n")
		p.Fprintf(w, "  --set oas.port=<port>                       Legacy API port (use --mwlegacy-port)\n")
		p.Fprintf(w, "  --set oas.credSetupFile=<path>               Credentials file (use --config)\n")
		p.Fprintf(w, "  --set oas.appFile=<path>                     App config file (deprecated)\n")
		p.Fprintf(w, "  --set mito.http.tls.certificatepath=<path>   TLS cert (use --tls-cert)\n")
		p.Fprintf(w, "  --set mito.http.tls.privatekeypath=<path>    TLS key (use --tls-key)\n")
		p.Fprintf(w, "  --set mito.http.tls.enabled=false            Disable TLS even if certs present\n\n")
		p.Fprintf(w, "Legacy Environment Variables:\n")
		p.Fprintf(w, "  MW_PORT                      Legacy API port (use OAUTH_MWLEGACY_PORT)\n")
		p.Fprintf(w, "  MW_CREDSETUPFILE             Credentials file (use OAUTH_CREDS_FILE)\n")
		p.Fprintf(w, "  MW_APPFILE                   App config file (deprecated)\n\n")
	}

	p.Fprintf(w, "Examples:\n")
	p.Fprintf(w, "  # Docker with mounted credentials\n")
	p.Fprintf(w, "  docker run -p 80:80 -p 30045:30045 \\\n")
	p.Fprintf(w, "    -v ./creds.yaml:/etc/oauth2-auth-code-proxy/creds.yaml:ro \\\n")
	p.Fprintf(w, "    -e OAUTH_CREDS_FILE=/etc/oauth2-auth-code-proxy/creds.yaml \\\n")
	p.Fprintf(w, "    oauth2-auth-code-proxy:latest\n\n")
	p.Fprintf(w, "  # Binary with HTTPS\n")
	p.Fprintf(w, "  oauth2-auth-code-proxy --config creds.yaml --tls-cert tls.crt --tls-key tls.key\n\n")

	p.Fprintf(w, "Set LANG environment variable for localized output (e.g., LANG=fr-FR).\n\n")

	p.Fprintf(w, "Copyright (c) 2020-2026, The MathWorks, Inc. All rights reserved.\n")
	p.Fprintf(w, "Governed by the MathWorks Cloud Reference Architecture License.\n")
	p.Fprintf(w, "See https://github.com/mathworks-ref-arch/container-images/oauth2-auth-code-proxy/blob/main/LICENSE.md\n")
}
