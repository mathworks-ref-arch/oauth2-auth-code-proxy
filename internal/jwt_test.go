// Copyright (c) 2020-2026, The MathWorks, Inc. All rights reserved.

package proxy

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJWTDecode_ValidToken(t *testing.T) {
	r := require.New(t)

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"123","email":"test@example.com","name":"Test"}`))
	sig := base64.RawURLEncoding.EncodeToString([]byte("signature-bytes"))

	token := strings.Join([]string{header, payload, sig}, ".")

	jt, err := jwtDecode(token)
	r.NoError(err)

	var claims map[string]any
	err = json.Unmarshal(jt.Payload, &claims)
	r.NoError(err)
	r.Equal("123", claims["sub"])
	r.Equal("test@example.com", claims["email"])
	r.Equal("Test", claims["name"])
}

func TestJWTDecode_InvalidFormat(t *testing.T) {
	r := require.New(t)

	_, err := jwtDecode("not.a.valid.token.too.many")
	r.Error(err)

	_, err = jwtDecode("onlyone")
	r.Error(err)
}

func TestJWTDecode_PaddingHandled(t *testing.T) {
	r := require.New(t)

	payload := `{"sub":"12345"}`
	encoded := base64.RawURLEncoding.EncodeToString([]byte(payload))
	header := base64.RawURLEncoding.EncodeToString([]byte(`{}`))
	sig := base64.RawURLEncoding.EncodeToString([]byte("sig"))

	token := strings.Join([]string{header, encoded, sig}, ".")
	jt, err := jwtDecode(token)
	r.NoError(err)

	var claims map[string]any
	r.NoError(json.Unmarshal(jt.Payload, &claims))
	r.Equal("12345", claims["sub"])
}
