package sentinel

import (
	"log"
	"strings"
	"time"
)


func logInfof(format string, args ...any) {
	log.Printf("[info] "+format, args...)
}

func logWarnf(format string, args ...any) {
	log.Printf("[warn] "+format, args...)
}

func logErrorf(format string, args ...any) {
	log.Printf("[error] "+format, args...)
}

// Run starts sentinel loop.
func Run(componentName string) {
	name := strings.TrimSpace(componentName)
	if name == "" {
		name = "tarinio-sentinel"
	}
	cfg := LoadConfig()
	st := loadState(cfg.StatePath)
	logInfof(
		"%s: started (enabled=%t ml_enabled=%t poll_interval=%s log_path=%s state_path=%s output_path=%s runtime_root=%s)",
		name,
		cfg.ModelEnabled,
		cfg.MLEnabled,
		cfg.PollInterval,
		cfg.LogPath,
		cfg.StatePath,
		cfg.OutputPath,
		cfg.RuntimeRoot,
	)
	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()
	var lastAdaptiveWrite time.Time
	var lastSuggestionWrite time.Time

	for {
		now := time.Now().UTC()
		effective := cfg
		if profile, ok := LoadRuntimeProfile(cfg.RuntimeRoot); ok {
			effective = ApplyRuntimeProfile(cfg, profile)
		}
		next, changed, err := processTick(effective, st, now)
		if err != nil {
			logErrorf("%s: tick failed: %v", name, err)
		} else {
			st = next
			if changed {
				if err := saveState(cfg.StatePath, st); err != nil {
					logErrorf("%s: save state failed: %v", name, err)
				}
			}
			if _, err := SaveAdaptiveIfChanged(cfg.OutputPath, effective, st, now, cfg.PublishInterval, &lastAdaptiveWrite); err != nil {
				logErrorf("%s: save adaptive output failed: %v", name, err)
			}
			if _, err := SaveSuggestionsIfChanged(cfg.SuggestionsOutputPath, st, now, cfg.PublishInterval, &lastSuggestionWrite); err != nil {
				logErrorf("%s: save suggestions output failed: %v", name, err)
			}
		}
		<-ticker.C
	}
}
