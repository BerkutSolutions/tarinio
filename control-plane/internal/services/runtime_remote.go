package services

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"waf/control-plane/internal/telemetry"
)

// HTTPReloadExecutor delegates runtime reload execution to the isolated runtime container.
type HTTPReloadExecutor struct {
	URL   string
	Token string
}

func (e HTTPReloadExecutor) Run(name string, args []string, workdir string) error {
	url := strings.TrimSpace(e.URL)
	candidates := runtimeEndpointCandidates(url, "http://127.0.0.1:8081/reload")
	client := &http.Client{Timeout: 10 * time.Second}
	var lastErr error
	for _, candidate := range candidates {
		req, err := http.NewRequest(http.MethodPost, candidate, nil)
		if err != nil {
			lastErr = err
			continue
		}
		setRuntimeAuthHeader(req, strings.TrimSpace(e.Token))

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		func() {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusNoContent {
				lastErr = nil
				return
			}
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			message := strings.TrimSpace(string(body))
			if message == "" {
				lastErr = fmt.Errorf("runtime reload endpoint returned %d", resp.StatusCode)
				return
			}
			lastErr = fmt.Errorf("runtime reload endpoint returned %d: %s", resp.StatusCode, message)
		}()
		if lastErr == nil {
			telemetry.Default().RecordRuntimeReload("runtime", "succeeded")
			return nil
		}
		// Runtime endpoint is reachable but returned an application-level failure.
		telemetry.Default().RecordRuntimeReload("runtime", "failed")
		return lastErr
	}
	if lastErr != nil {
		telemetry.Default().RecordRuntimeReload("runtime", "error")
		return lastErr
	}
	telemetry.Default().RecordRuntimeReload("runtime", "error")
	return fmt.Errorf("runtime reload endpoint is not reachable")
}
