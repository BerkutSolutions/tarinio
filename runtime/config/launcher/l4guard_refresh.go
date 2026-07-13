package main

import (
	"crypto/sha256"
	"os"
	"strings"
)

func l4GuardConfigFingerprint() [sha256.Size]byte {
	hash := sha256.New()
	for _, path := range []string{
		"/etc/waf/l4guard/config.json",
		l4GuardAdaptivePath(),
	} {
		_, _ = hash.Write([]byte(path))
		if content, err := os.ReadFile(path); err == nil {
			_, _ = hash.Write(content)
		}
	}
	for _, key := range []string{
		"WAF_L4_GUARD_ENABLED", "WAF_L4_GUARD_CHAIN_MODE", "WAF_L4_GUARD_CONN_LIMIT",
		"WAF_L4_GUARD_RATE_PER_SECOND", "WAF_L4_GUARD_RATE_BURST", "WAF_L4_GUARD_PORTS",
		"WAF_L4_GUARD_TARGET", "WAF_L4_GUARD_DESTINATION_IP",
	} {
		_, _ = hash.Write([]byte(key + "=" + os.Getenv(key)))
	}
	var fingerprint [sha256.Size]byte
	copy(fingerprint[:], hash.Sum(nil))
	return fingerprint
}

func l4GuardAdaptivePath() string {
	if path := strings.TrimSpace(os.Getenv("WAF_L4_GUARD_ADAPTIVE_PATH")); path != "" {
		return path
	}
	return "/etc/waf/l4guard-adaptive/adaptive.json"
}
