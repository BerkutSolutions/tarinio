package sentinel

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"strings"
	"time"
)

type mlLogisticArtifact struct {
	Version string             `json:"version"`
	Type    string             `json:"type"`
	Bias    float64            `json:"bias"`
	Weights map[string]float64 `json:"weights"`
}

type mlLogisticModel struct {
	version string
	bias    float64
	weights map[string]float64
}

type mlCacheState struct {
	path    string
	modTime time.Time
	model   *mlLogisticModel
}

var mlCache mlCacheState

func loadMLModel(path string) (*mlLogisticModel, error) {
	artifactPath := strings.TrimSpace(path)
	if artifactPath == "" {
		return nil, errors.New("empty MODEL_ML_ARTIFACT_PATH")
	}
	info, err := os.Stat(artifactPath)
	if err != nil {
		return nil, err
	}
	if mlCache.model != nil && mlCache.path == artifactPath && info.ModTime().Equal(mlCache.modTime) {
		return mlCache.model, nil
	}
	raw, err := os.ReadFile(artifactPath)
	if err != nil {
		return nil, err
	}
	var artifact mlLogisticArtifact
	if err := json.Unmarshal(raw, &artifact); err != nil {
		return nil, err
	}
	model, err := compileMLModel(artifact)
	if err != nil {
		return nil, err
	}
	mlCache = mlCacheState{
		path:    artifactPath,
		modTime: info.ModTime(),
		model:   model,
	}
	return model, nil
}

func compileMLModel(artifact mlLogisticArtifact) (*mlLogisticModel, error) {
	modelType := strings.ToLower(strings.TrimSpace(artifact.Type))
	if modelType == "" {
		modelType = "logistic_regression"
	}
	if modelType != "logistic_regression" {
		return nil, errors.New("unsupported ml model type: " + modelType)
	}
	if len(artifact.Weights) == 0 {
		return nil, errors.New("empty weights in ml artifact")
	}
	weights := make(map[string]float64, len(artifact.Weights))
	for key, value := range artifact.Weights {
		name := strings.TrimSpace(key)
		if name == "" {
			continue
		}
		weights[name] = value
	}
	if len(weights) == 0 {
		return nil, errors.New("ml artifact has no valid feature weights")
	}
	version := strings.TrimSpace(artifact.Version)
	if version == "" {
		version = "unversioned"
	}
	return &mlLogisticModel{
		version: version,
		bias:    artifact.Bias,
		weights: weights,
	}, nil
}

func inferMLWeight(stats *ipStat, cfg Config, model *mlLogisticModel) (float64, string, bool) {
	if model == nil || stats == nil || stats.Count <= 0 {
		return 0, "", false
	}
	probability := inferMLProbability(stats, model)
	if probability < cfg.MLMinProbability {
		return 0, "", false
	}
	weight := clamp(probability*cfg.MLMaxWeight, 0, cfg.MLMaxWeight)
	if weight <= 0 {
		return 0, "", false
	}
	return weight, model.version, true
}

func inferMLProbability(stats *ipStat, model *mlLogisticModel) float64 {
	features := buildMLFeatures(stats)
	score := model.bias
	for feature, weight := range model.weights {
		score += features[feature] * weight
	}
	return sigmoid(score)
}

func buildMLFeatures(stats *ipStat) map[string]float64 {
	if stats == nil || stats.Count <= 0 {
		return map[string]float64{}
	}
	total := float64(stats.Count)
	notFoundRatio := float64(stats.NotFound) / total
	blockedRatio := float64(stats.Blocked) / total
	return map[string]float64{
		"rps_per_ip":            float64(stats.Count),
		"not_found_ratio":       notFoundRatio,
		"blocked_ratio":         blockedRatio,
		"scanner_hits":          float64(stats.ScannerHits),
		"suspicious_ua_hits":    float64(stats.SuspiciousUAHits),
		"unique_paths":          float64(len(stats.UniquePaths)),
		"cross_site_spread":     float64(len(stats.Sites)),
		"unique_paths_per_req":  float64(len(stats.UniquePaths)) / total,
		"scanner_ratio":         float64(stats.ScannerHits) / total,
		"suspicious_ua_ratio":   float64(stats.SuspiciousUAHits) / total,
	}
}

func sigmoid(x float64) float64 {
	if x > 20 {
		return 1
	}
	if x < -20 {
		return 0
	}
	return 1 / (1 + math.Exp(-x))
}
