package easysiteprofiles

import "testing"

// tab05_ban_escalation_test.go — тесты вкладки 5: Эскалация банов
// Покрывает: нормализацию, валидацию BanEscalationEnabled/Scope/StagesSeconds.

// --- Нормализация scope ---

func TestBanEscalation_Normalize_ScopeDefault(t *testing.T) {
	p := DefaultProfile("site-a")
	p.SecurityBehaviorAndLimits.BanEscalationScope = ""
	out := normalizeProfile(p)
	if out.SecurityBehaviorAndLimits.BanEscalationScope != "all_sites" {
		t.Fatalf("expected default scope all_sites, got %q", out.SecurityBehaviorAndLimits.BanEscalationScope)
	}
}

func TestBanEscalation_Normalize_ScopeUpperCase(t *testing.T) {
	p := DefaultProfile("site-a")
	p.SecurityBehaviorAndLimits.BanEscalationScope = "ALL_SITES"
	out := normalizeProfile(p)
	if out.SecurityBehaviorAndLimits.BanEscalationScope != "all_sites" {
		t.Fatalf("expected lowercase all_sites after normalize, got %q", out.SecurityBehaviorAndLimits.BanEscalationScope)
	}
}

func TestBanEscalation_Normalize_CurrentSite(t *testing.T) {
	p := DefaultProfile("site-a")
	p.SecurityBehaviorAndLimits.BanEscalationScope = "current_site"
	out := normalizeProfile(p)
	if out.SecurityBehaviorAndLimits.BanEscalationScope != "current_site" {
		t.Fatalf("expected current_site scope preserved, got %q", out.SecurityBehaviorAndLimits.BanEscalationScope)
	}
}

// --- Нормализация stages ---

func TestBanEscalation_Normalize_StagesDefault(t *testing.T) {
	p := DefaultProfile("site-a")
	p.SecurityBehaviorAndLimits.BanEscalationStagesSeconds = nil
	out := normalizeProfile(p)
	if len(out.SecurityBehaviorAndLimits.BanEscalationStagesSeconds) == 0 {
		t.Fatalf("expected default stages after normalize, got empty")
	}
}

func TestBanEscalation_Normalize_StagesDeduped(t *testing.T) {
	p := DefaultProfile("site-a")
	// normalizeBanEscalationStages должна убрать дубликаты нулей
	p.SecurityBehaviorAndLimits.BanEscalationStagesSeconds = []int{300, 0, 0}
	out := normalizeProfile(p)
	stages := out.SecurityBehaviorAndLimits.BanEscalationStagesSeconds
	zeroCount := 0
	for _, s := range stages {
		if s == 0 {
			zeroCount++
		}
	}
	if zeroCount > 1 {
		t.Fatalf("expected at most one permanent (0) stage after normalize, got %v", stages)
	}
}

// --- Валидация scope ---

func TestBanEscalation_Validate_InvalidScope(t *testing.T) {
	p := DefaultProfile("site-a")
	p.SecurityBehaviorAndLimits.BanEscalationEnabled = true
	p.SecurityBehaviorAndLimits.BanEscalationScope = "invalid_scope"
	p.SecurityBehaviorAndLimits.BanEscalationStagesSeconds = []int{300, 0}
	if err := validateProfile(p); err == nil {
		t.Fatal("expected error for invalid ban escalation scope")
	}
}

func TestBanEscalation_Validate_AllSitesScope_Valid(t *testing.T) {
	p := DefaultProfile("site-a")
	p.SecurityBehaviorAndLimits.BanEscalationEnabled = true
	p.SecurityBehaviorAndLimits.BanEscalationScope = "all_sites"
	p.SecurityBehaviorAndLimits.BanEscalationStagesSeconds = []int{300, 86400, 0}
	if err := validateProfile(p); err != nil {
		t.Fatalf("expected no error for valid all_sites scope, got: %v", err)
	}
}

func TestBanEscalation_Validate_CurrentSiteScope_Valid(t *testing.T) {
	p := DefaultProfile("site-a")
	p.SecurityBehaviorAndLimits.BanEscalationEnabled = true
	p.SecurityBehaviorAndLimits.BanEscalationScope = "current_site"
	p.SecurityBehaviorAndLimits.BanEscalationStagesSeconds = []int{300, 0}
	if err := validateProfile(p); err != nil {
		t.Fatalf("expected no error for valid current_site scope, got: %v", err)
	}
}

// --- Валидация stages ---

func TestBanEscalation_Validate_EmptyStages_WhenEnabled(t *testing.T) {
	p := DefaultProfile("site-a")
	p.SecurityBehaviorAndLimits.BanEscalationEnabled = true
	p.SecurityBehaviorAndLimits.BanEscalationStagesSeconds = []int{}
	if err := validateProfile(p); err == nil {
		t.Fatal("expected error for empty stages when escalation enabled")
	}
}

func TestBanEscalation_Validate_TooManyStages(t *testing.T) {
	p := DefaultProfile("site-a")
	p.SecurityBehaviorAndLimits.BanEscalationEnabled = true
	stages := make([]int, 13) // > 12
	for i := range stages {
		stages[i] = 300
	}
	p.SecurityBehaviorAndLimits.BanEscalationStagesSeconds = stages
	if err := validateProfile(p); err == nil {
		t.Fatal("expected error for more than 12 stages")
	}
}

func TestBanEscalation_Validate_NegativeStage(t *testing.T) {
	p := DefaultProfile("site-a")
	p.SecurityBehaviorAndLimits.BanEscalationEnabled = true
	p.SecurityBehaviorAndLimits.BanEscalationStagesSeconds = []int{300, -1, 0}
	if err := validateProfile(p); err == nil {
		t.Fatal("expected error for negative stage value")
	}
}

func TestBanEscalation_Validate_PermanentNotLast(t *testing.T) {
	p := DefaultProfile("site-a")
	p.SecurityBehaviorAndLimits.BanEscalationEnabled = true
	// 0 (permanent) стоит не последним
	p.SecurityBehaviorAndLimits.BanEscalationStagesSeconds = []int{0, 300, 86400}
	if err := validateProfile(p); err == nil {
		t.Fatal("expected error when permanent stage (0) is not last")
	}
}

func TestBanEscalation_Validate_PermanentAsLastStage_Valid(t *testing.T) {
	p := DefaultProfile("site-a")
	p.SecurityBehaviorAndLimits.BanEscalationEnabled = true
	p.SecurityBehaviorAndLimits.BanEscalationStagesSeconds = []int{60, 300, 3600, 0}
	if err := validateProfile(p); err != nil {
		t.Fatalf("expected no error when permanent is last stage, got: %v", err)
	}
}

func TestBanEscalation_Validate_SingleFiniteStage_Valid(t *testing.T) {
	p := DefaultProfile("site-a")
	p.SecurityBehaviorAndLimits.BanEscalationEnabled = true
	p.SecurityBehaviorAndLimits.BanEscalationStagesSeconds = []int{300}
	if err := validateProfile(p); err != nil {
		t.Fatalf("expected no error for single finite stage, got: %v", err)
	}
}

func TestBanEscalation_Validate_MaxStages_Valid(t *testing.T) {
	p := DefaultProfile("site-a")
	p.SecurityBehaviorAndLimits.BanEscalationEnabled = true
	stages := make([]int, 12)
	for i := range stages {
		stages[i] = (i + 1) * 60
	}
	p.SecurityBehaviorAndLimits.BanEscalationStagesSeconds = stages
	if err := validateProfile(p); err != nil {
		t.Fatalf("expected no error for exactly 12 stages, got: %v", err)
	}
}

// --- Disabled не требует stages ---

func TestBanEscalation_Validate_Disabled_NoStagesRequired(t *testing.T) {
	p := DefaultProfile("site-a")
	p.SecurityBehaviorAndLimits.BanEscalationEnabled = false
	p.SecurityBehaviorAndLimits.BanEscalationStagesSeconds = nil
	if err := validateProfile(p); err != nil {
		t.Fatalf("expected no error when escalation disabled with no stages, got: %v", err)
	}
}
