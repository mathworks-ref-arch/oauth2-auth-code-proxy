// Copyright (c) 2020-2026, The MathWorks, Inc. All rights reserved.

package proxy

import (
	"encoding/base64"
	"errors"
	"strings"
)

type jwtToken struct {
	Header    []byte
	Payload   []byte
	Signature []byte
}

func jwtDecode(token string) (jwtToken, error) {
	sections := strings.Split(token, ".")
	if len(sections) != 3 {
		return jwtToken{}, errors.New("parsing JWT token failed")
	}

	header, err := jwtBase64DecodeWithPadding(sections[0])
	if err != nil {
		return jwtToken{}, err
	}
	payload, err := jwtBase64DecodeWithPadding(sections[1])
	if err != nil {
		return jwtToken{}, err
	}
	signature, err := jwtBase64DecodeWithPadding(sections[2])
	if err != nil {
		return jwtToken{}, err
	}

	return jwtToken{
		Header:    header,
		Payload:   payload,
		Signature: signature,
	}, nil
}

func jwtBase64DecodeWithPadding(str string) ([]byte, error) {
	if l := len(str) % 4; l > 0 {
		str += strings.Repeat("=", 4-l)
	}
	return base64.URLEncoding.DecodeString(str)
}
