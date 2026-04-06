package appcompat

import (
	"fmt"
	"strings"
	"time"
)

type ModuleCompat struct {
	ModuleID string `json:"module_id"`
	Status   Status `json:"status"`

	TitleI18nKey   string `json:"title_i18n_key"`
	DetailsI18nKey string `json:"details_i18n_key"`

	ExpectedSchemaVersion   int `json:"expected_schema_version"`
	AppliedSchemaVersion    int `json:"applied_schema_version"`
	ExpectedBehaviorVersion int `json:"expected_behavior_version"`
	AppliedBehaviorVersion  int `json:"applied_behavior_version"`

	HasPartialAdapt bool   `json:"has_partial_adapt"`
	HasFullReset    bool   `json:"has_full_reset"`
	DangerLevel     string `json:"danger_level"`

	LastError string `json:"last_error,omitempty"`
}

type Report struct {
	NowUTC time.Time      `json:"now_utc"`
	Items  []ModuleCompat `json:"items"`
}

func BuildReport(now time.Time, pendingByModule map[string][]string) Report {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	specs := DefaultRegistry()
	items := make([]ModuleCompat, 0, len(specs))
	for _, spec := range specs {
		status := StatusOK
		lastError := ""
		pendingDirs := pendingByModule[spec.ModuleID]
		if len(pendingDirs) > 0 {
			status = StatusNeedsAttention
			lastError = fmt.Sprintf("legacy storage layout detected for %s: %s", spec.ModuleID, strings.Join(pendingDirs, ", "))
		}
		items = append(items, ModuleCompat{
			ModuleID:                spec.ModuleID,
			Status:                  status,
			TitleI18nKey:            spec.TitleI18nKey,
			DetailsI18nKey:          spec.DetailsI18nKey,
			ExpectedSchemaVersion:   spec.ExpectedSchemaVersion,
			AppliedSchemaVersion:    spec.ExpectedSchemaVersion,
			ExpectedBehaviorVersion: spec.ExpectedBehaviorVersion,
			AppliedBehaviorVersion:  spec.ExpectedBehaviorVersion,
			HasPartialAdapt:         spec.HasPartialAdapt,
			HasFullReset:            spec.HasFullReset,
			DangerLevel:             spec.DangerLevel,
			LastError:               lastError,
		})
	}
	return Report{
		NowUTC: now.UTC(),
		Items:  items,
	}
}
