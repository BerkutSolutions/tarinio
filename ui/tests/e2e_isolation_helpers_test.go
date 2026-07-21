package tests

import (
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

var e2eIDSequence uint64

// e2eUniqueID prevents one test's persisted object from becoming another
// test's candidate input when a suite is retried or extended.
func e2eUniqueID(t *testing.T, prefix string) string {
	t.Helper()
	name := strings.ToLower(t.Name())
	name = strings.NewReplacer("/", "-", "_", "-", " ", "-").Replace(name)
	name = strings.Trim(name, "-")
	if len(name) > 22 {
		name = name[len(name)-22:]
	}
	return fmt.Sprintf("%s-%s-%x-%d", prefix, name, time.Now().UnixNano(), atomic.AddUint64(&e2eIDSequence, 1))
}
