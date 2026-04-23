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
	if url == "" {
		url = "http://127.0.0.1:8081/reload"
	}

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	setRuntimeAuthHeader(req, strings.TrimSpace(e.Token))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		telemetry.Default().RecordRuntimeReload("runtime", "error")
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		telemetry.Default().RecordRuntimeReload("runtime", "failed")
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		message := strings.TrimSpace(string(body))
		if message == "" {
			return fmt.Errorf("runtime reload endpoint returned %d", resp.StatusCode)
		}
		return fmt.Errorf("runtime reload endpoint returned %d: %s", resp.StatusCode, message)
	}
	telemetry.Default().RecordRuntimeReload("runtime", "succeeded")
	return nil
}
