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
	var lastErr error
	for _, candidate := range candidates {
		for attempt := 0; attempt < 2; attempt++ {
			lastErr, reachable := e.reloadOnce(candidate)
			if lastErr == nil {
				telemetry.Default().RecordRuntimeReload("runtime", "succeeded")
				return nil
			}
			if reachable {
				// A response from runtime is a definitive validation/reload failure.
				telemetry.Default().RecordRuntimeReload("runtime", "failed")
				return lastErr
			}
			if attempt == 0 {
				time.Sleep(750 * time.Millisecond)
			}
		}
	}
	if lastErr != nil {
		telemetry.Default().RecordRuntimeReload("runtime", "error")
		return lastErr
	}
	telemetry.Default().RecordRuntimeReload("runtime", "error")
	return fmt.Errorf("runtime reload endpoint is not reachable")
}

// reloadOnce returns reachable=false only for a transport-level interruption.
// That can happen while launcher atomically replaces nginx during a revision.
func (e HTTPReloadExecutor) reloadOnce(endpoint string) (error, bool) {
	req, err := http.NewRequest(http.MethodPost, endpoint, nil)
	if err != nil {
		return err, false
	}
	setRuntimeAuthHeader(req, strings.TrimSpace(e.Token))
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return err, false
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		return nil, true
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	message := strings.TrimSpace(string(body))
	if message == "" {
		return fmt.Errorf("runtime reload endpoint returned %d", resp.StatusCode), true
	}
	return fmt.Errorf("runtime reload endpoint returned %d: %s", resp.StatusCode, message), true
}
