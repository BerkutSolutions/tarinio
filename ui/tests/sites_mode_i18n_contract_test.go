package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSitesModeI18N_RussianNoteMatchesAvailableTabs(t *testing.T) {
	modeFiles := []string{
		filepath.Join("..", "app", "static", "js", "pages", "sites.detail-shell.js"),
		filepath.Join("..", "app", "static", "js", "pages", "sites.view-io.js"),
	}

	for _, file := range modeFiles {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		content := string(raw)
		if strings.Contains(content, `ctx.t("sites.mode.advanced")`) {
			t.Fatalf("stale advanced mode tab marker still present in %s", file)
		}
		if !strings.Contains(content, `data-mode-tab="raw"`) {
			t.Fatalf("expected raw mode tab marker in %s", file)
		}
		if strings.Contains(content, `data-mode-tab="raw" disabled`) {
			t.Fatalf("raw mode tab unexpectedly disabled in %s", file)
		}
	}

	ruFile := filepath.Join("..", "app", "static", "i18n", "ru.json")
	raw, err := os.ReadFile(ruFile)
	if err != nil {
		t.Fatalf("read %s: %v", ruFile, err)
	}
	content := string(raw)

	for _, marker := range []string{
		`"sites.mode.note": "Простой режим — это пошаговый редактор. Используйте сырой режим, когда нужен прямой env-стиль контроля над той же моделью сервиса."`,
		`"sites.mode.raw": "Сырой"`,
	} {
		if !strings.Contains(content, marker) {
			t.Fatalf("missing marker %q in %s", marker, ruFile)
		}
	}

	if strings.Contains(content, `"sites.mode.note": "Сейчас доступен только простой режим.`) {
		t.Fatalf("stale russian sites.mode.note still claims only easy mode is available in %s", ruFile)
	}
}