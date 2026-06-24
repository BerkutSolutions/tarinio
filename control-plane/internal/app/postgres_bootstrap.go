package app

import (
	"strings"
	"time"
)

const (
	postgresBootstrapWaitInterval = 1500 * time.Millisecond
	postgresBootstrapWaitTimeout  = 45 * time.Second
)

func waitForPostgresReady(ping func() error) error {
	return waitForPostgresReadyWithInterval(ping, postgresBootstrapWaitTimeout, postgresBootstrapWaitInterval)
}

func waitForPostgresReadyWithInterval(ping func() error, timeout time.Duration, interval time.Duration) error {
	if ping == nil {
		return nil
	}
	deadline := time.Now().Add(timeout)
	for {
		err := ping()
		if err == nil {
			return nil
		}
		if !isRetryablePostgresStartupError(err) || time.Now().After(deadline) {
			return err
		}
		time.Sleep(interval)
	}
}

func isRetryablePostgresStartupError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(message, "connection refused"):
		return true
	case strings.Contains(message, "the database system is shutting down"):
		return true
	case strings.Contains(message, "terminating connection due to administrator command"):
		return true
	case strings.Contains(message, "sqlstate 57p03"):
		return true
	case strings.Contains(message, "sqlstate 57p01"):
		return true
	case strings.Contains(message, "no route to host"):
		return true
	default:
		return false
	}
}
