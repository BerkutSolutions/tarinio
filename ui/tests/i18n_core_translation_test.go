package tests

import (
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestI18NCoreRuNotEnglish(t *testing.T) {
	ru := mustLoadLang(t, filepath.Join("..", "app", "static", "i18n", "ru.json"))
	en := mustLoadLang(t, filepath.Join("..", "app", "static", "i18n", "en.json"))

	keys := []string{
		"app.dashboard",
		"app.sites",
		"app.antiddos",
		"app.tls",
		"app.requests",
		"app.events",
		"app.bans",
		"app.jobs",
		"app.administration",
		"app.activity",
		"app.settings",
		"app.profile",
		"app.version",
		"app.section.dashboard.desc",
		"app.section.sites.desc",
		"app.section.antiddos.desc",
		"app.section.tls.desc",
		"app.section.requests.desc",
		"app.section.events.desc",
		"app.section.bans.desc",
		"app.section.jobs.desc",
		"app.section.administration.desc",
		"app.section.activity.desc",
		"app.section.settings.desc",
		"app.section.profile.desc",
		"profile.title",
		"profile.subtitle",
		"profile.field.username",
		"profile.field.roles",
		"profile.field.permissions",
		"settings.tabs.general",
		"settings.tabs.about",
		"common.save",
		"common.cancel",
		"common.close",
		"common.delete",
	}

	var untranslated []string
	for _, key := range keys {
		rv := strings.TrimSpace(ru[key])
		ev := strings.TrimSpace(en[key])
		if rv == "" || ev == "" {
			untranslated = append(untranslated, key+" (missing)")
			continue
		}
		if strings.EqualFold(rv, ev) {
			untranslated = append(untranslated, key)
		}
	}

	if len(untranslated) > 0 {
		sort.Strings(untranslated)
		t.Fatalf("core ru keys are untranslated: %v", sample(untranslated))
	}
}
