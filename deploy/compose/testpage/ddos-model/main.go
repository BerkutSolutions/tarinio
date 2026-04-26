package main

import (
	"time"

	"waf/internal/sentinel"
)

type modelConfig = sentinel.Config
type runtimeProfile = sentinel.RuntimeProfile
type state = sentinel.State
type record = sentinel.Record
type adaptiveOutput = sentinel.AdaptiveOutput

func main() {
	sentinel.Run("tarinio-sentinel")
}

func applyRuntimeProfile(base modelConfig, profile runtimeProfile) modelConfig {
	return sentinel.ApplyRuntimeProfile(base, profile)
}

func effectiveEmergencyThresholds(cfg modelConfig) (int, int, int) {
	return sentinel.EffectiveEmergencyThresholds(cfg)
}

func saveAdaptive(path string, cfg modelConfig, st state, now time.Time) error {
	return sentinel.SaveAdaptive(path, cfg, st, now)
}

func makeScoreKey(site, ip string) string {
	return sentinel.MakeScoreKey(site, ip)
}
