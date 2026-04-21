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
	if err := application.RunStartupSelfTest(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "control-plane startup self-test failed: %v\n", err)
		os.Exit(1)
	}

	if application.DevFastStartBootstrapper != nil {
		go func() {
			if err := application.DevFastStartBootstrapper.Run(context.Background()); err != nil {
				fmt.Fprintf(os.Stderr, "control-plane dev fast start: %v\n", err)
			}
		}()
	}
	if application.Coordinator != nil && application.Coordinator.Enabled() {
		fmt.Fprintf(os.Stdout, "control-plane ha enabled: node=%s\n", application.Coordinator.NodeID())
	}

	if err := application.HTTPServer.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintf(os.Stderr, "control-plane server: %v\n", err)
		os.Exit(1)
	}
}
