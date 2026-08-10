package proxy

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrintHelp_English(t *testing.T) {
	a := assert.New(t)
	var buf bytes.Buffer

	PrintHelp(&buf, "en-US", "test")
	out := buf.String()

	a.Contains(out, "OAuth2 Token Proxy (v")
	a.Contains(out, "OAuth2 token exchange proxy for desktop clients")
	a.Contains(out, "Usage:")
	a.Contains(out, "--config <path>")
	a.Contains(out, "Options:")
	a.Contains(out, "Environment Variables:")
	a.Contains(out, "OAUTH_CREDS_FILE")
	a.Contains(out, "Examples:")
	a.Contains(out, "docker run")
	a.Contains(out, "Set LANG environment variable")
	a.Contains(out, "Copyright (c) 2020-2026, The MathWorks, Inc.")
	a.Contains(out, "Governed by the MathWorks Cloud Reference Architecture License.")
	a.Contains(out, "LICENSE.md")
}

func TestPrintHelp_French(t *testing.T) {
	a := assert.New(t)
	var buf bytes.Buffer

	PrintHelp(&buf, "fr-FR", "test")
	out := buf.String()

	a.Contains(out, "Proxy de jetons OAuth2 (v")
	a.Contains(out, "Proxy d'échange de jetons OAuth2 pour les clients de bureau")
	a.Contains(out, "Utilisation :")
	a.Contains(out, "Options :")
	a.Contains(out, "Variables d'environnement :")
	a.Contains(out, "Exemples :")
	a.Contains(out, "Définissez la variable d'environnement LANG")
	a.Contains(out, "Tous droits réservés.")
	a.Contains(out, "Régi par la licence MathWorks Cloud Reference Architecture.")
}

func TestPrintHelp_FallbackToEnglish(t *testing.T) {
	a := assert.New(t)
	var buf bytes.Buffer

	PrintHelp(&buf, "ja-JP", "test")
	out := buf.String()

	a.Contains(out, "OAuth2 Token Proxy (v")
	a.Contains(out, "Usage:")
}

func TestPrintHelp_EmptyLang(t *testing.T) {
	a := assert.New(t)
	var buf bytes.Buffer

	PrintHelp(&buf, "", "test")
	out := buf.String()

	a.Contains(out, "OAuth2 Token Proxy (v")
}
