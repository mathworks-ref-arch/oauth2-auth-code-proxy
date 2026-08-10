// Copyright (c) 2020-2026, The MathWorks, Inc. All rights reserved.

package domain

import (
	"encoding/json"
	"net/http"
)

func WriteJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func WriteFault(w http.ResponseWriter, faultID, message string) {
	WriteJSON(w, FaultResponse{
		MWType:    "core/Fault",
		FaultID:   []string{faultID},
		MessageID: message,
	})
}
