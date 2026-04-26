package sentinel

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompileMLModelRejectsUnsupportedType(t *testing.T) {
	_, err := compileMLModel(mlLogisticArtifact{
		Type:    "gbdt",
		Weights: map[string]float64{"rps_per_ip": 0.1},
	})
	if err == nil {
		t.Fatal("expected unsupported type error")
	}
}

func TestInferMLWeightAppliesProbabilityThreshold(t *testing.T) {
	model := &mlLogisticModel{
		version: "v1",
		bias:    -2.0,
		weights: map[string]float64{
			"rps_per_ip":       0.04,
			"scanner_hits":     0.6,
			"suspicious_ua_hits": 0.3,
		},
	}
	cfg := Config{
		MLMinProbability: 0.60,
		MLMaxWeight:      1.5,
	}
	stats := &ipStat{
		Count:            120,
		ScannerHits:      4,
		SuspiciousUAHits: 2,
		UniquePaths:      map[string]struct{}{"/.env": {}, "/wp-admin": {}},
		Sites:            map[string]struct{}{"app-a": {}, "app-b": {}},
	}
	weight, version, ok := inferMLWeight(stats, cfg, model)
	if !ok {
		t.Fatal("expected ml inference to produce a weight")
	}
	if weight <= 0 || weight > cfg.MLMaxWeight {
		t.Fatalf("unexpected ml weight: %f", weight)
	}
	if version != "v1" {
		t.Fatalf("unexpected model version: %q", version)
	}
}

func TestLoadMLModelUsesArtifactAndCache(t *testing.T) {
	mlCache = mlCacheState{}
	path := filepath.Join(t.TempDir(), "ml-model.json")
	if err := os.WriteFile(path, []byte(`{
  "version": "2026-04-26",
  "type": "logistic_regression",
  "bias": -1.5,
  "weights": {
    "rps_per_ip": 0.02,
    "scanner_hits": 0.8
  }
}`), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	first, err := loadMLModel(path)
	if err != nil {
		t.Fatalf("load model first: %v", err)
	}
	second, err := loadMLModel(path)
	if err != nil {
		t.Fatalf("load model second: %v", err)
	}
	if first != second {
		t.Fatal("expected cached model instance to be reused")
	}
}

func TestDeriveSignalWeightsIncludesCrossSiteSpread(t *testing.T) {
	signals := deriveSignalWeights(&ipStat{
		Count:       40,
		NotFound:    20,
		ScannerHits: 3,
		UniquePaths: map[string]struct{}{
			"/.env":      {},
			"/wp-admin":  {},
			"/phpmyadmin": {},
		},
		Sites: map[string]struct{}{
			"site-a": {},
			"site-b": {},
			"site-c": {},
		},
	})
	if signals["signal_cross_site_spread"] <= 0 {
		t.Fatalf("expected cross-site signal, got %#v", signals)
	}
}
