package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

const crsTrustedDigestsEnv = "WAF_CRS_TRUSTED_SHA256"

// builtInCRSTrustedDigests is reviewed and shipped with the WAF release.
// Each digest covers the exact CRS release asset selected by crsManager, not
// an untrusted value from the runtime release metadata.
var builtInCRSTrustedDigests = map[string]string{
	"4.28.0": "fca67fe46adafeeee61b9d1a03f38c25b9b2a799577df03fa51d99589e6d03b9",
}

func defaultCRSTrustedDigests() map[string]string {
	values := make(map[string]string, len(builtInCRSTrustedDigests))
	for version, digest := range builtInCRSTrustedDigests {
		values[version] = digest
	}
	return values
}

// loadCRSTrustedDigests reads version=digest pairs from protected runtime
// configuration. Built-in release pins and protected overrides take precedence.
// Otherwise, the updater uses the SHA-256 digest published for the exact
// official GitHub release asset after validating its repository-owned URL.
func loadCRSTrustedDigests() (map[string]string, error) {
	values := defaultCRSTrustedDigests()
	for _, entry := range strings.Split(strings.TrimSpace(os.Getenv(crsTrustedDigestsEnv)), ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		version, digest, ok := strings.Cut(entry, "=")
		version = normalizeVersion(version)
		digest = strings.ToLower(strings.TrimSpace(digest))
		if !ok || version == "" || len(digest) != 64 {
			return nil, fmt.Errorf("invalid %s entry", crsTrustedDigestsEnv)
		}
		if _, err := hex.DecodeString(digest); err != nil {
			return nil, fmt.Errorf("invalid %s digest for CRS %s", crsTrustedDigestsEnv, version)
		}
		values[version] = digest
	}
	return values, nil
}
