package services

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// HTTPReloadExecutor delegates runtime reload execution to the isolated runtime container.
type HTTPReloadExecutor struct {
	URL string
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

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("runtime reload endpoint returned %d", resp.StatusCode)
	}
	return nil
}

