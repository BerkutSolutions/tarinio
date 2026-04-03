package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	"waf/control-plane/internal/app"
	"waf/control-plane/internal/config"
)

func main() {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "control-plane config: %v\n", err)
		os.Exit(1)
	}

	application, err := app.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "control-plane bootstrap: %v\n", err)
		os.Exit(1)
	}

	if application.DevFastStartBootstrapper != nil {
		go func() {
			if err := application.DevFastStartBootstrapper.Run(context.Background()); err != nil {
				fmt.Fprintf(os.Stderr, "control-plane dev fast start: %v\n", err)
			}
		}()
	}

	if err := application.HTTPServer.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintf(os.Stderr, "control-plane server: %v\n", err)
		os.Exit(1)
	}
}
