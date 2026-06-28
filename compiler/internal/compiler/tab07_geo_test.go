package compiler

import (
	"strings"
	"testing"
)

// tab07_geo_test.go — тесты вкладки 7: Гео-фильтрация и временные окна
// Покрывает: BlacklistCountry, WhitelistCountry (уже в tab04),
// GeoTimeWindows: snippet в site.conf, map-артефакт в HTTP-конфиге,
// block/allow action, hour range, days of week, exception bypass,
// невалидные окна игнорируются.

// --- GeoTimeWindow: snippet появляется в site.conf ---

func TestGeo_TimeWindow_SnippetInSiteConf(t *testing.T) {
	conf := mustRenderSiteConf(t, "geo-tw-1", EasyProfileInput{
		SiteID:         "geo-tw-1",
		SecurityMode:   "block",
		AllowedMethods: []string{"GET"},
		MaxClientSize:  "10m",
		GeoTimeWindows: []GeoTimeWindowInput{
			{Countries: []string{"CN"}, Action: "block", HoursStart: 9, HoursEnd: 18},
		},
	})
	if !strings.Contains(conf, "geo time-window enforcement") {
		t.Fatalf("expected geo time-window snippet in site.conf, got:\n%s", conf)
	}
	// Snippet содержит переменную country guard (не сам код страны)
	if !strings.Contains(conf, "waf_geo_tw_") {
		t.Fatalf("expected waf_geo_tw_ variable in geo snippet, got:\n%s", conf)
	}
}

// --- GeoTimeWindow: block action → return 403 ---

func TestGeo_TimeWindow_BlockAction_Returns403(t *testing.T) {
	conf := mustRenderSiteConf(t, "geo-tw-block", EasyProfileInput{
		SiteID:         "geo-tw-block",
		SecurityMode:   "block",
		AllowedMethods: []string{"GET"},
		MaxClientSize:  "10m",
		GeoTimeWindows: []GeoTimeWindowInput{
			{Countries: []string{"DE"}, Action: "block", HoursStart: 0, HoursEnd: 8},
		},
	})
	if !strings.Contains(conf, "return 403") {
		t.Fatalf("expected return 403 for geo time-window block action, got:\n%s", conf)
	}
}

// --- GeoTimeWindow: allow action → нет return 403 в snippet ---

func TestGeo_TimeWindow_AllowAction_No403(t *testing.T) {
	snippet := buildGeoTimeWindowServerSnippet("geo-allow", []GeoTimeWindowInput{
		{Countries: []string{"RU"}, Action: "allow", HoursStart: 9, HoursEnd: 18},
	}, "$waf_easy_exception_guard")
	if strings.Contains(snippet, "return 403") {
		t.Fatalf("did not expect return 403 for allow action snippet, got:\n%s", snippet)
	}
	// Allow action — snippet может быть пустым (нет блокировки) или содержать переменные
	// Главное — нет return 403
}

// --- GeoTimeWindow: hour range в snippet ---

func TestGeo_TimeWindow_HourRange_InSnippet(t *testing.T) {
	snippet := buildGeoTimeWindowServerSnippet("geo-hrs", []GeoTimeWindowInput{
		{Countries: []string{"US"}, Action: "block", HoursStart: 9, HoursEnd: 17},
	}, "$waf_easy_exception_guard")
	// Часы 09..16 должны быть в regex паттерне
	for _, h := range []string{"09", "10", "11", "16"} {
		if !strings.Contains(snippet, h) {
			t.Fatalf("expected hour %s in geo time-window snippet, got:\n%s", h, snippet)
		}
	}
	// Час 17 (exclusive) не должен быть
	if strings.Contains(snippet, "17") {
		t.Fatalf("did not expect exclusive hour 17 in snippet, got:\n%s", snippet)
	}
}

// --- GeoTimeWindow: days of week фильтрация ---

func TestGeo_TimeWindow_DaysOfWeek_InSnippet(t *testing.T) {
	snippet := buildGeoTimeWindowServerSnippet("geo-days", []GeoTimeWindowInput{
		{Countries: []string{"FR"}, Action: "block", DaysOfWeek: []int{1, 2, 3, 4, 5}, HoursStart: 9, HoursEnd: 18},
	}, "$waf_easy_exception_guard")
	// Snippet должен содержать переменные и return 403
	if !strings.Contains(snippet, "waf_geo_tw_") {
		t.Fatalf("expected waf_geo_tw_ variable in days-of-week snippet, got:\n%s", snippet)
	}
	if !strings.Contains(snippet, "return 403") {
		t.Fatalf("expected return 403 in block days-of-week snippet, got:\n%s", snippet)
	}
}

// --- GeoTimeWindow: exception guard bypass ---

func TestGeo_TimeWindow_ExceptionGuard_Bypass(t *testing.T) {
	snippet := buildGeoTimeWindowServerSnippet("geo-exc", []GeoTimeWindowInput{
		{Countries: []string{"CN"}, Action: "block", HoursStart: 0, HoursEnd: 23},
	}, "$waf_easy_exception_guard")
	// Guard строится как "${exception}:${active}" → "0:1" блокирует
	if !strings.Contains(snippet, "0:1") {
		t.Fatalf("expected exception guard pattern 0:1 in snippet, got:\n%s", snippet)
	}
}

// --- GeoTimeWindow: map-артефакт в HTTP-конфиге ---

func TestGeo_TimeWindow_MapArtifact_Generated(t *testing.T) {
	artifacts, err := RenderEasyArtifacts(
		[]SiteInput{{ID: "geo-map", Enabled: true, PrimaryHost: "geo-map.example.com", ListenHTTP: true}},
		[]EasyProfileInput{{
			SiteID:         "geo-map",
			SecurityMode:   "block",
			AllowedMethods: []string{"GET"},
			MaxClientSize:  "10m",
			GeoTimeWindows: []GeoTimeWindowInput{
				{Countries: []string{"JP", "KR"}, Action: "block", HoursStart: 2, HoursEnd: 6},
			},
		}},
	)
	if err != nil {
		t.Fatalf("RenderEasyArtifacts: %v", err)
	}
	byPath := artifactsByPath(artifacts)
	geoArt, ok := byPath["nginx/geo-timewindow/geo-map.conf"]
	if !ok {
		t.Fatalf("expected artifact nginx/geo-timewindow/geo-map.conf, got paths: %v", func() []string {
			var keys []string
			for k := range byPath {
				keys = append(keys, k)
			}
			return keys
		}())
	}
	geoConf := string(geoArt.Content)
	if !strings.Contains(geoConf, `"JP" 1`) {
		t.Fatalf("expected JP country mapping in geo map artifact, got:\n%s", geoConf)
	}
	if !strings.Contains(geoConf, `"KR" 1`) {
		t.Fatalf("expected KR country mapping in geo map artifact, got:\n%s", geoConf)
	}
	if !strings.Contains(geoConf, "map $time_iso8601") {
		t.Fatalf("expected hour map in geo map artifact, got:\n%s", geoConf)
	}
}

// --- GeoTimeWindow: HTTP map-конфиг содержит hour map ---

func TestGeo_TimeWindow_HttpConf_HourMap(t *testing.T) {
	conf := buildGeoTimeWindowHttpConf("geo-http", []GeoTimeWindowInput{
		{Countries: []string{"CN"}, Action: "block", HoursStart: 9, HoursEnd: 18},
	})
	if !strings.Contains(conf, "map $time_iso8601") {
		t.Fatalf("expected map $time_iso8601 in http conf, got:\n%s", conf)
	}
	if !strings.Contains(conf, "waf_geo_geo_http_hour") {
		t.Fatalf("expected hour variable in http conf, got:\n%s", conf)
	}
}

// --- GeoTimeWindow: HTTP map-конфиг содержит country map ---

func TestGeo_TimeWindow_HttpConf_CountryMap(t *testing.T) {
	conf := buildGeoTimeWindowHttpConf("geo-cmap", []GeoTimeWindowInput{
		{Countries: []string{"CN", "RU"}, Action: "block", HoursStart: 0, HoursEnd: 23},
	})
	if !strings.Contains(conf, "map $waf_country_code") {
		t.Fatalf("expected map $waf_country_code in http conf, got:\n%s", conf)
	}
	if !strings.Contains(conf, `"CN" 1`) {
		t.Fatalf("expected CN country mapping in http conf, got:\n%s", conf)
	}
	if !strings.Contains(conf, `"RU" 1`) {
		t.Fatalf("expected RU country mapping in http conf, got:\n%s", conf)
	}
}

// --- GeoTimeWindow: невалидное окно игнорируется ---

func TestGeo_TimeWindow_InvalidWindow_Ignored(t *testing.T) {
	snippet := buildGeoTimeWindowServerSnippet("geo-inv", []GeoTimeWindowInput{
		// Невалидное: HoursStart >= HoursEnd — будет отброшено
		{Countries: []string{"CN"}, Action: "block", HoursStart: 18, HoursEnd: 9},
		// Валидное — должно попасть в snippet
		{Countries: []string{"DE"}, Action: "block", HoursStart: 9, HoursEnd: 18},
	}, "$waf_easy_exception_guard")
	// Валидное окно → snippet не пустой
	if !strings.Contains(snippet, "geo time-window enforcement") {
		t.Fatalf("expected snippet from valid window, got:\n%s", snippet)
	}
	// Невалидное окно → только один window-блок (один _0_ индекс, нет _1_)
	if strings.Contains(snippet, "_1_") {
		t.Fatalf("did not expect second window (_1_) from invalid input, got:\n%s", snippet)
	}
}

// --- GeoTimeWindow: пустой список → нет snippet ---

func TestGeo_TimeWindow_Empty_NoSnippet(t *testing.T) {
	conf := mustRenderSiteConf(t, "geo-empty", EasyProfileInput{
		SiteID:         "geo-empty",
		SecurityMode:   "block",
		AllowedMethods: []string{"GET"},
		MaxClientSize:  "10m",
		GeoTimeWindows: nil,
	})
	if strings.Contains(conf, "geo time-window enforcement") {
		t.Fatalf("did not expect geo snippet when GeoTimeWindows nil, got:\n%s", conf)
	}
}

// --- GeoTimeWindow: несколько окон ---

func TestGeo_TimeWindow_MultipleWindows(t *testing.T) {
	snippet := buildGeoTimeWindowServerSnippet("geo-multi", []GeoTimeWindowInput{
		{Countries: []string{"CN"}, Action: "block", HoursStart: 0, HoursEnd: 8},
		{Countries: []string{"RU"}, Action: "block", HoursStart: 20, HoursEnd: 23},
	}, "$waf_easy_exception_guard")
	// Два валидных окна → два индекса _0_ и _1_
	if !strings.Contains(snippet, "_0_") {
		t.Fatalf("expected first window _0_ in multi-window snippet, got:\n%s", snippet)
	}
	if !strings.Contains(snippet, "_1_") {
		t.Fatalf("expected second window _1_ in multi-window snippet, got:\n%s", snippet)
	}
}

// --- Validate: невалидный action ---

func TestGeo_Validate_InvalidAction(t *testing.T) {
	err := ValidateGeoTimeWindow(GeoTimeWindowInput{
		Countries: []string{"CN"}, Action: "deny", HoursStart: 9, HoursEnd: 18,
	})
	if err == nil {
		t.Fatal("expected error for invalid action deny")
	}
}

// --- Validate: невалидный код страны ---

func TestGeo_Validate_InvalidCountryCode(t *testing.T) {
	err := ValidateGeoTimeWindow(GeoTimeWindowInput{
		Countries: []string{"CHINA"}, Action: "block", HoursStart: 9, HoursEnd: 18,
	})
	if err == nil {
		t.Fatal("expected error for invalid country code CHINA")
	}
}

// --- Validate: hours_start >= hours_end ---

func TestGeo_Validate_HoursStartGEHoursEnd(t *testing.T) {
	err := ValidateGeoTimeWindow(GeoTimeWindowInput{
		Countries: []string{"CN"}, Action: "block", HoursStart: 18, HoursEnd: 9,
	})
	if err == nil {
		t.Fatal("expected error when hours_start >= hours_end")
	}
}

// --- Validate: day_of_week вне диапазона ---

func TestGeo_Validate_InvalidDayOfWeek(t *testing.T) {
	err := ValidateGeoTimeWindow(GeoTimeWindowInput{
		Countries: []string{"CN"}, Action: "block", DaysOfWeek: []int{7}, HoursStart: 9, HoursEnd: 18,
	})
	if err == nil {
		t.Fatal("expected error for day_of_week=7")
	}
}
