// Copyright (c) 2020-2026, The MathWorks, Inc. All rights reserved.

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mathworks-ref-arch/container-images/oauth2-auth-code-proxy/internal/app"
)

func main() {
	if err := app.Run(context.Background(), os.Args, os.Getenv, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
