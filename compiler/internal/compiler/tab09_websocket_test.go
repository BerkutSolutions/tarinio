package compiler

import (
	"strings"
	"testing"
)

// tab09_websocket_test.go — тесты вкладки 9: WebSocket инспекция
// Покрывает: UseWSInspection включён/выключен, snippet в site.conf,
// WSBlockPatterns, WSMaxMessageBytes, WSRateMsgPerSec, валидация паттернов.

// --- UseWSInspection: snippet в site.conf ---

func TestWebSocket_Inspection_SnippetInSiteConf(t *testing.T) {
	conf := mustRenderSiteConf(t, "ws-insp", EasyProfileInput{
		SiteID:                "ws-insp",
		SecurityMode:          "block",
		AllowedMethods:        []string{"GET"},
		MaxClientSize:         "10m",
		ReverseProxyWebsocket: true,
		WSInspection: WSInspectionInput{
			UseWSInspection: true,
			WSBlockPatterns: []string{`DROP TABLE`},
		},
	})
	// WS snippet содержит Lua-блок
	if !strings.Contains(conf, "DROP TABLE") {
		t.Fatalf("expected WS block pattern in site.conf when WSInspection enabled, got:\n%s", conf)
	}
}

// --- UseWSInspection: disabled → нет snippet ---

func TestWebSocket_Inspection_Disabled_NoSnippet(t *testing.T) {
	conf := mustRenderSiteConf(t, "ws-insp-off", EasyProfileInput{
		SiteID:                "ws-insp-off",
		SecurityMode:          "block",
		AllowedMethods:        []string{"GET"},
		MaxClientSize:         "10m",
		ReverseProxyWebsocket: true,
		WSInspection: WSInspectionInput{
			UseWSInspection: false,
			WSBlockPatterns: []string{`DROP TABLE`},
		},
	})
	if strings.Contains(conf, "waf_ws") {
		t.Fatalf("did not expect waf_ws snippet when UseWSInspection=false, got:\n%s", conf)
	}
}

// --- WSBlockPatterns в snippet ---

func TestWebSocket_BlockPattern_InSnippet(t *testing.T) {
	snippet := buildWSInspectionServerSnippet("ws-pat", WSInspectionInput{
		UseWSInspection: true,
		WSBlockPatterns: []string{`<script>`, `DROP TABLE`},
	})
	if !strings.Contains(snippet, `<script>`) {
		t.Fatalf("expected pattern <script> in WS snippet, got:\n%s", snippet)
	}
	if !strings.Contains(snippet, `DROP TABLE`) {
		t.Fatalf("expected pattern DROP TABLE in WS snippet, got:\n%s", snippet)
	}
}

// --- WSMaxMessageBytes в snippet ---

func TestWebSocket_MaxMessageBytes_InSnippet(t *testing.T) {
	snippet := buildWSInspectionServerSnippet("ws-maxb", WSInspectionInput{
		UseWSInspection:   true,
		WSBlockPatterns:   []string{`test`},
		WSMaxMessageBytes: 65536,
	})
	if !strings.Contains(snippet, "65536") {
		t.Fatalf("expected max message bytes 65536 in WS snippet, got:\n%s", snippet)
	}
}

// --- WSRateMsgPerSec в snippet ---

func TestWebSocket_RateMsgPerSec_InSnippet(t *testing.T) {
	snippet := buildWSInspectionServerSnippet("ws-rate", WSInspectionInput{
		UseWSInspection: true,
		WSBlockPatterns: []string{`test`},
		WSRateMsgPerSec: 100,
	})
	if !strings.Contains(snippet, "100") {
		t.Fatalf("expected rate 100 in WS snippet, got:\n%s", snippet)
	}
}

// --- Пустые паттерны → нет snippet ---

func TestWebSocket_NoPatterns_NoSnippet(t *testing.T) {
	snippet := buildWSInspectionServerSnippet("ws-nopat", WSInspectionInput{
		UseWSInspection: true,
		WSBlockPatterns: nil,
	})
	if snippet != "" {
		t.Fatalf("expected empty snippet when no patterns and no limits, got:\n%s", snippet)
	}
}

// --- Валидация: невалидный regex ---

func TestWebSocket_Validate_InvalidPattern(t *testing.T) {
	err := ValidateWSInspection(WSInspectionInput{
		UseWSInspection: true,
		WSBlockPatterns: []string{`valid`, `[invalid`},
	})
	if err == nil {
		t.Fatal("expected error for invalid regex pattern [invalid")
	}
}

// --- Валидация: валидные паттерны ---

func TestWebSocket_Validate_ValidPatterns(t *testing.T) {
	err := ValidateWSInspection(WSInspectionInput{
		UseWSInspection: true,
		WSBlockPatterns: []string{`DROP TABLE`, `(?i)select.*from`, `<script>`},
	})
	if err != nil {
		t.Fatalf("expected no error for valid patterns, got: %v", err)
	}
}

// --- Нормализация: дубликаты паттернов удаляются ---

func TestWebSocket_Normalize_DedupPatterns(t *testing.T) {
	result := normalizeWSBlockPatterns([]string{"pat1", "pat1", "pat2", "  pat2  "})
	if len(result) != 2 {
		t.Fatalf("expected 2 unique patterns after dedup, got %d: %v", len(result), result)
	}
}

// --- Нормализация: пустые строки удаляются ---

func TestWebSocket_Normalize_EmptyRemoved(t *testing.T) {
	result := normalizeWSBlockPatterns([]string{"  ", "", "pat1", ""})
	if len(result) != 1 {
		t.Fatalf("expected 1 pattern after removing empty, got %d: %v", len(result), result)
	}
}
