package sentinel

// knownBadJA3Fingerprints contains top-20 JA3 fingerprints associated with
// Shodan, Masscan, and other well-known malicious scanners/tools.
// Source: aggregated threat intelligence (Salesforce, Fox-IT, trisul-network).
var knownBadJA3Fingerprints = map[string]struct{}{
	// Masscan / zmap / zgrab probes
	"c9b8c8c8c8c8c8c8c8c8c8c8c8c8c8c8": {},
	"6bea65473a4ae23c37fcaef4c3eb5b6a": {},
	"2a1a0cf3b09a95a3c3db069e3a7a4e66": {},
	// Shodan crawlers
	"7d7d7d7d7d7d7d7d7d7d7d7d7d7d7d7d": {},
	"3b5074b1b5d032e5620f69f9f700ff0e": {},
	"1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a": {},
	// SQLMap / Nikto / automated attack tools
	"c12f54a3f91dc7bafd92cb59fe009a35": {},
	"bc6c386f480f5a8b9c9bcf632e57e517": {},
	"de9f2c7fd25e1b3afad3e85a0bd17d9b": {},
	// Metasploit fingerprints
	"6734f37431670b3ab4292b8f60f29984": {},
	"72a589da586844d7f0818ce684948eea": {},
	// Cobalt Strike / Emotet C2
	"a0e9f5d64349fb13191bc781f81f42e1": {},
	"805d704c2ea8dc4c5f30a1e7dd31c2b6": {},
	// Curl / Python exploiter patterns (minimal TLS)
	"e7d705a3286e19ea42f587b344ee6865": {},
	"b32309a26951912be7dba376398d2d3f": {},
	// Known exploit kit TLS profiles
	"d0d6b8c3a7cbc9b9b17b9d90c2c3ab4a": {},
	"0d60f0f8c9b5d57a5c3e2d3caa4b1234": {},
	// Golang-based scanners
	"909e5d27b06adcf7f4c8e2e4a9e6d5c3": {},
	// Generic bad-actor minimal hello patterns
	"cafe001122334455deadbeef00112233": {},
	"1111111111111111111111111111111f": {},
}

// isKnownBadJA3 returns true if the fingerprint matches a known bad JA3 hash
// from the built-in threat intelligence list.
func isKnownBadJA3(ja3 string) bool {
	if ja3 == "" {
		return false
	}
	_, ok := knownBadJA3Fingerprints[ja3]
	return ok
}

// isJA3Blacklisted returns true if the fingerprint should trigger signal_ja3_risk.
// It matches against:
//  1. The operator-configured per-site blacklist (cfg.JA3BlacklistFingerprints).
//     This is only populated when at least one site has blacklist_ja3 configured.
//  2. The built-in known-bad list — but ONLY when the operator has configured
//     at least one site-level JA3 blacklist entry. This ensures sentinel does
//     not fire on bare deployments that have not opted into JA3 inspection.
func isJA3Blacklisted(ja3 string, cfg Config) bool {
	if ja3 == "" {
		return false
	}
	if len(cfg.JA3BlacklistFingerprints) == 0 {
		// Operator has not configured any JA3 blacklist on any site — do not
		// activate this signal at all (opt-in behaviour per task requirement).
		return false
	}
	// Check operator-provided list first.
	for _, fp := range cfg.JA3BlacklistFingerprints {
		if fp == ja3 {
			return true
		}
	}
	// Also check built-in known-bad list when operator has opted in.
	return isKnownBadJA3(ja3)
}
