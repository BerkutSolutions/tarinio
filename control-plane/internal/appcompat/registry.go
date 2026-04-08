package appcompat

type Status string

const (
	StatusOK             Status = "ok"
	StatusNeedsAttention Status = "needs_attention"
	StatusNeedsReinit    Status = "needs_reinit"
	StatusBroken         Status = "broken"
)

type ModuleSpec struct {
	ModuleID string

	TitleI18nKey   string
	DetailsI18nKey string

	ExpectedSchemaVersion   int
	ExpectedBehaviorVersion int

	HasPartialAdapt bool
	HasFullReset    bool
	DangerLevel     string
}

// ContractVersion is used by tests and installer checks to force explicit updates
// when compatibility logic changes.
const ContractVersion = "2026-04-08-healthcheck-v2"

func DefaultRegistry() []ModuleSpec {
	return []ModuleSpec{
		{ModuleID: "dashboard", TitleI18nKey: "compat.module.dashboard", DetailsI18nKey: "compat.details.dashboard", ExpectedSchemaVersion: 1, ExpectedBehaviorVersion: 1, HasPartialAdapt: true, HasFullReset: true, DangerLevel: "low"},
		{ModuleID: "sites", TitleI18nKey: "compat.module.sites", DetailsI18nKey: "compat.details.sites", ExpectedSchemaVersion: 1, ExpectedBehaviorVersion: 1, HasPartialAdapt: true, HasFullReset: true, DangerLevel: "critical"},
		{ModuleID: "antiddos", TitleI18nKey: "compat.module.antiddos", DetailsI18nKey: "compat.details.antiddos", ExpectedSchemaVersion: 1, ExpectedBehaviorVersion: 1, HasPartialAdapt: true, HasFullReset: true, DangerLevel: "high"},
		{ModuleID: "owaspcrs", TitleI18nKey: "compat.module.owaspcrs", DetailsI18nKey: "compat.details.owaspcrs", ExpectedSchemaVersion: 1, ExpectedBehaviorVersion: 1, HasPartialAdapt: true, HasFullReset: true, DangerLevel: "medium"},
		{ModuleID: "tls", TitleI18nKey: "compat.module.tls", DetailsI18nKey: "compat.details.tls", ExpectedSchemaVersion: 1, ExpectedBehaviorVersion: 1, HasPartialAdapt: true, HasFullReset: true, DangerLevel: "critical"},
		{ModuleID: "requests", TitleI18nKey: "compat.module.requests", DetailsI18nKey: "compat.details.requests", ExpectedSchemaVersion: 1, ExpectedBehaviorVersion: 1, HasPartialAdapt: true, HasFullReset: true, DangerLevel: "medium"},
		{ModuleID: "events", TitleI18nKey: "compat.module.events", DetailsI18nKey: "compat.details.events", ExpectedSchemaVersion: 1, ExpectedBehaviorVersion: 1, HasPartialAdapt: true, HasFullReset: true, DangerLevel: "low"},
		{ModuleID: "bans", TitleI18nKey: "compat.module.bans", DetailsI18nKey: "compat.details.bans", ExpectedSchemaVersion: 1, ExpectedBehaviorVersion: 1, HasPartialAdapt: true, HasFullReset: true, DangerLevel: "high"},
		{ModuleID: "administration", TitleI18nKey: "compat.module.administration", DetailsI18nKey: "compat.details.administration", ExpectedSchemaVersion: 1, ExpectedBehaviorVersion: 1, HasPartialAdapt: true, HasFullReset: true, DangerLevel: "medium"},
		{ModuleID: "activity", TitleI18nKey: "compat.module.activity", DetailsI18nKey: "compat.details.activity", ExpectedSchemaVersion: 1, ExpectedBehaviorVersion: 1, HasPartialAdapt: true, HasFullReset: true, DangerLevel: "low"},
		{ModuleID: "settings", TitleI18nKey: "compat.module.settings", DetailsI18nKey: "compat.details.settings", ExpectedSchemaVersion: 1, ExpectedBehaviorVersion: 1, HasPartialAdapt: true, HasFullReset: true, DangerLevel: "medium"},
		{ModuleID: "profile", TitleI18nKey: "compat.module.profile", DetailsI18nKey: "compat.details.profile", ExpectedSchemaVersion: 1, ExpectedBehaviorVersion: 1, HasPartialAdapt: true, HasFullReset: true, DangerLevel: "low"},
	}
}
