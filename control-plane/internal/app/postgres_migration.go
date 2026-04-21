package app

import (
	"path/filepath"

	"waf/control-plane/internal/storage"
)

func migrateLegacyStateToPostgres(backend storage.Backend, revisionStoreDir string) error {
	docs := []struct {
		key        string
		legacyPath string
	}{
		{key: "revisions/revisions.json", legacyPath: filepath.Join(revisionStoreDir, "revisions", "revisions.json")},
		{key: "events/events.json", legacyPath: filepath.Join(revisionStoreDir, "events", "events.json")},
		{key: "audits/audit_events.json", legacyPath: filepath.Join(revisionStoreDir, "audits", "audit_events.json")},
		{key: "jobs/jobs.json", legacyPath: filepath.Join(revisionStoreDir, "jobs", "jobs.json")},
		{key: "roles/roles.json", legacyPath: filepath.Join(revisionStoreDir, "roles", "roles.json")},
		{key: "users/users.json", legacyPath: filepath.Join(revisionStoreDir, "users", "users.json")},
		{key: "sessions/sessions.json", legacyPath: filepath.Join(revisionStoreDir, "sessions", "sessions.json")},
		{key: "passkeys/passkeys.json", legacyPath: filepath.Join(revisionStoreDir, "passkeys", "passkeys.json")},
		{key: "sites/sites.json", legacyPath: filepath.Join(revisionStoreDir, "sites", "sites.json")},
		{key: "upstreams/upstreams.json", legacyPath: filepath.Join(revisionStoreDir, "upstreams", "upstreams.json")},
		{key: "certificates/certificates.json", legacyPath: filepath.Join(revisionStoreDir, "certificates", "certificates.json")},
		{key: "certificatematerials/materials.json", legacyPath: filepath.Join(revisionStoreDir, "certificate-materials", "materials.json")},
		{key: "tlsconfigs/tls_configs.json", legacyPath: filepath.Join(revisionStoreDir, "tlsconfigs", "tls_configs.json")},
		{key: "wafpolicies/waf_policies.json", legacyPath: filepath.Join(revisionStoreDir, "wafpolicies", "waf_policies.json")},
		{key: "accesspolicies/access_policies.json", legacyPath: filepath.Join(revisionStoreDir, "accesspolicies", "access_policies.json")},
		{key: "ratelimitpolicies/rate_limit_policies.json", legacyPath: filepath.Join(revisionStoreDir, "ratelimitpolicies", "rate_limit_policies.json")},
		{key: "easysiteprofiles/easy_site_profiles.json", legacyPath: filepath.Join(revisionStoreDir, "easysiteprofiles", "easy_site_profiles.json")},
		{key: "antiddos/anti_ddos_settings.json", legacyPath: filepath.Join(revisionStoreDir, "antiddos", "anti_ddos_settings.json")},
		{key: "tls-auto-renew/settings.json", legacyPath: filepath.Join(revisionStoreDir, "tls-auto-renew", "settings.json")},
		{key: "settings/runtime_settings.json", legacyPath: filepath.Join(revisionStoreDir, "settings", "runtime_settings.json")},
	}
	for _, item := range docs {
		if err := storage.MigrateLegacyDocument(backend, item.key, item.legacyPath); err != nil {
			return err
		}
	}
	if err := storage.MigrateLegacyDocumentDir(backend, "snapshots", filepath.Join(revisionStoreDir, "revision-snapshots")); err != nil {
		return err
	}
	if err := storage.MigrateLegacyBlobDir(backend, "certificate-materials/files", filepath.Join(revisionStoreDir, "certificate-materials", "files")); err != nil {
		return err
	}
	if err := storage.MigrateLegacyBlobDir(backend, "revision-snapshots/files", filepath.Join(revisionStoreDir, "revision-snapshots", "files")); err != nil {
		return err
	}
	return nil
}
