package tests

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func e2eEnvironmentValue(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func e2eAttackerIP() string { return e2eEnvironmentValue("WAF_E2E_ATTACKER_IP", "172.30.0.30") }

func e2eL4AttackerIP() string { return e2eEnvironmentValue("WAF_E2E_L4_ATTACKER_IP", "172.30.0.31") }

var e2eIDSequence uint64

// e2eUniqueID prevents one test's persisted object from becoming another
// test's candidate input when a suite is retried or extended.
func e2eUniqueID(t *testing.T, prefix string) string {
	t.Helper()
	name := strings.ToLower(t.Name())
	name = strings.NewReplacer("/", "-", "_", "-", " ", "-").Replace(name)
	name = strings.Trim(name, "-")
	if len(name) > 8 {
		name = name[len(name)-8:]
	}
	// Site IDs flow into Nginx shared-memory zone names. Keep them compact
	// enough for Nginx while retaining per-test uniqueness.
	return fmt.Sprintf("%s-%s-%x-%d", prefix, name, time.Now().UnixNano()&0xffffff, atomic.AddUint64(&e2eIDSequence, 1))
}
