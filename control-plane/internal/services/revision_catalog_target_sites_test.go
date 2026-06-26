package services

import (
	"reflect"
	"testing"
)

func TestNarrowScopeByTargets_NoTargetsKeepsScope(t *testing.T) {
	scope := revisionSiteScope{
		ChangedSites: []RevisionCatalogSite{
			{SiteID: "alpha"},
			{SiteID: "beta"},
		},
		ActiveSiteIDs: []string{"alpha"},
	}
	siteRef := map[string]RevisionCatalogSite{
		"alpha": {SiteID: "alpha", PrimaryHost: "alpha.example.com"},
		"beta":  {SiteID: "beta", PrimaryHost: "beta.example.com"},
	}
	sites, active := narrowScopeByTargets(scope, nil, siteRef)
	if !reflect.DeepEqual(sites, scope.ChangedSites) {
		t.Fatalf("expected unchanged scope sites, got %+v", sites)
	}
	if !reflect.DeepEqual(active, scope.ActiveSiteIDs) {
		t.Fatalf("expected unchanged active ids, got %+v", active)
	}
}

func TestNarrowScopeByTargets_FiltersToExplicitTargets(t *testing.T) {
	scope := revisionSiteScope{
		ChangedSites: []RevisionCatalogSite{
			{SiteID: "alpha", PrimaryHost: "alpha.example.com"},
			{SiteID: "beta", PrimaryHost: "beta.example.com"},
		},
		ActiveSiteIDs: []string{"alpha", "beta"},
	}
	siteRef := map[string]RevisionCatalogSite{
		"alpha": scope.ChangedSites[0],
		"beta":  scope.ChangedSites[1],
	}
	sites, active := narrowScopeByTargets(scope, []string{"alpha"}, siteRef)
	if len(sites) != 1 || sites[0].SiteID != "alpha" {
		t.Fatalf("expected only alpha in narrowed sites, got %+v", sites)
	}
	if len(active) != 1 || active[0] != "alpha" {
		t.Fatalf("expected only alpha in narrowed active ids, got %+v", active)
	}
}

// On a fresh baseline revision the diff lists every site as "changed".
// narrowScopeByTargets must still scope the revision to the explicit target
// the operator passed at compile time, otherwise the UI shows "this revision
// touched sites the operator never edited".
func TestNarrowScopeByTargets_BaselineRevisionScopesToTargetOnly(t *testing.T) {
	scope := revisionSiteScope{
		ChangedSites: []RevisionCatalogSite{
			{SiteID: "sentry.hantico.ru", PrimaryHost: "sentry.hantico.ru"},
			{SiteID: "waf.hantico.ru", PrimaryHost: "waf.hantico.ru"},
		},
		ActiveSiteIDs: nil,
	}
	siteRef := map[string]RevisionCatalogSite{
		"sentry.hantico.ru": scope.ChangedSites[0],
		"waf.hantico.ru":    scope.ChangedSites[1],
	}
	sites, active := narrowScopeByTargets(scope, []string{"sentry.hantico.ru"}, siteRef)
	if len(sites) != 1 || sites[0].SiteID != "sentry.hantico.ru" {
		t.Fatalf("baseline revision must still scope to the explicit target, got %+v", sites)
	}
	if len(active) != 0 {
		t.Fatalf("expected no active sites for non-active revision, got %+v", active)
	}
}

// If the operator names a site that doesn't appear in the diff at all (edge
// case: revision diff says "nothing changed for that site" but operator
// pressed save on it), still surface the site so the UI doesn't lie about
// scope.
func TestNarrowScopeByTargets_AddsMissingTargetFromSiteRef(t *testing.T) {
	scope := revisionSiteScope{
		ChangedSites:  []RevisionCatalogSite{},
		ActiveSiteIDs: nil,
	}
	siteRef := map[string]RevisionCatalogSite{
		"alpha": {SiteID: "alpha", PrimaryHost: "alpha.example.com"},
	}
	sites, _ := narrowScopeByTargets(scope, []string{"alpha"}, siteRef)
	if len(sites) != 1 || sites[0].SiteID != "alpha" || sites[0].PrimaryHost != "alpha.example.com" {
		t.Fatalf("expected explicit target injected from siteRef, got %+v", sites)
	}
}

func TestNormalizeTargetSiteIDs_TrimDedupSort(t *testing.T) {
	out := normalizeTargetSiteIDs([]string{" Alpha ", "alpha", "", "beta"})
	want := []string{"alpha", "beta"}
	if !reflect.DeepEqual(out, want) {
		t.Fatalf("expected %+v, got %+v", want, out)
	}
}
