// Copyright (c) 2020-2026, The MathWorks, Inc. All rights reserved.

package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
)

func getIdentifiersFromToken(ctx context.Context, client HTTPClient, token *oauth2.Token, app AppConfig) (string, string, string) {
	var uniqueIdentifier, userEmail, userName string

	if idToken, ok := token.Extra("id_token").(string); ok {
		if jt, err := jwtDecode(idToken); err == nil {
			userInfo := map[string]any{}
			if err := json.Unmarshal(jt.Payload, &userInfo); err == nil {
				uniqueIdentifier, _ = userInfo["sub"].(string)        //nolint:errcheck
				userEmail, _ = userInfo[app.IDTokenEmailKey].(string) //nolint:errcheck
				userName, _ = userInfo[app.IDTokenNameKey].(string)   //nolint:errcheck
				if userEmail != "" && userName == "" {
					userName = userEmail
				}
			}
		}
	}

	if userEmail == "" || userName == "" {
		uniqueIdentifier, userEmail, userName = makeAPICallForAccountInfo(ctx, client, token, app)
	}
	if uniqueIdentifier == "" {
		uniqueIdentifier = userEmail
	}
	return uniqueIdentifier, userEmail, userName
}

func makeAPICallForAccountInfo(ctx context.Context, client HTTPClient, token *oauth2.Token, app AppConfig) (string, string, string) {
	temp, err := getAccountInfoFromOAuth(ctx, client, token.AccessToken, app.Host, app.TestingURL, app.IDTokenEmailKey)
	if err != nil {
		return "unknown", "unknown", "unknown"
	}

	userEmail, ok := temp[app.EmailKey].(string)
	if !ok {
		return "unknown", "unknown", "unknown"
	}
	userName, ok := temp[app.NameKey].(string)
	if !ok {
		userName = userEmail
	}
	uniqueID, ok := temp["sub"].(string)
	if !ok {
		uniqueID = fmt.Sprintf("%v", temp["sub"])
	}

	return uniqueID, userEmail, userName
}

func getAccountInfoFromOAuth(ctx context.Context, client HTTPClient, accessToken, host, url, primaryKey string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+accessToken)
	req.Header.Add("Host", host)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}

	data := map[string]any{}
	if err = json.Unmarshal(body, &data); err != nil {
		listData := []map[string]any{}
		err = json.Unmarshal(body, &listData)
		if err != nil {
			return nil, err
		}
		for _, ld := range listData {
			if val, ok := ld[primaryKey].(bool); ok && val {
				data = ld
				break
			}
		}
	}
	return data, nil
}
