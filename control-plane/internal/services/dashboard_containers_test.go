package services

import "testing"

func TestClassifyContainerLogIssue_IgnoresBenignOpenSearchStartupNoise(t *testing.T) {
	cases := []string{
		`[2026-04-22T17:33:58,531][WARN ][stderr ] [21ed019821fd] WARNING: Use --enable-native-access=ALL-UNNAMED to avoid a warning for this module`,
		`[2026-04-22T17:33:58,531][WARN ][stderr ] [21ed019821fd] WARNING: java.lang.foreign.Linker::downcallHandle has been called by the unnamed module`,
		`[2026-04-22T17:33:58,531][WARN ][stderr ] [21ed019821fd] WARNING: A restricted method in java.lang.foreign.Linker has been called`,
		`[2026-04-22T17:33:56,195][INFO ][o.o.n.Node ] [21ed019821fd] JVM arguments [-XX:+ShowCodeDetailsInExceptionMessages, -XX:ErrorFile=logs/hs_err_pid%p.log, -Dopensearch.path.home=/usr/share/opensearch]`,
		`[2026-04-22T17:39:11,177][WARN ][o.o.c.r.a.AllocationService] [f689172c038f] Falling back to single shard assignment since batch mode disable or multiple custom allocators set`,
		`WARNING: System::setSecurityManager will be removed in a future release`,
		`WARNING: A terminally deprecated method in java.lang.System has been called`,
		`[2026-04-22T17:39:10,792][WARN ][o.o.s.SecurityAnalyticsPlugin] [f689172c038f] Failed to initialize LogType config index and builtin log types`,
		`[2026-04-22T17:39:10,790][WARN ][o.o.o.i.ObservabilityIndex] [f689172c038f] message: index [.opensearch-observability/urMbmYjyRdOUP6V1UoeFSw] already exists`,
		`[2026-04-22T17:39:10,724][WARN ][o.o.p.c.s.h.ConfigOverridesClusterSettingHandler] [f689172c038f] Config override setting update called with empty string. Ignoring.`,
		`[2026-04-22T17:39:10,077][WARN ][o.o.g.DanglingIndicesState] [f689172c038f] gateway.auto_import_dangling_indices is disabled, dangling indices will not be automatically detected or imported and must be managed manually`,
		`[2026-04-22T17:39:09,384][WARN ][o.o.s.p.SQLPlugin ] [f689172c038f] Master key is a required config for using create and update datasource APIs. Please set plugins.query.datasources.encryption.masterkey config in opensearch.yml in all the cluster nodes.`,
		`[2026-04-22T17:39:06,788][WARN ][o.o.s.OpenSearchSecurityPlugin] [f689172c038f] OpenSearch Security plugin installed but disabled. This can expose your configuration (including passwords) to the public.`,
		`WARNING: Please consider reporting this to the maintainers of org.opensearch.bootstrap.Security`,
		`WARNING: System::setSecurityManager has been called by org.opensearch.bootstrap.Security (file:/usr/share/opensearch/lib/opensearch-2.18.0.jar)`,
		`WARNING: COMPAT locale provider will be removed in a future release`,
		`WARNING: Please consider reporting this to the maintainers of org.opensearch.bootstrap.OpenSearch`,
		`WARNING: System::setSecurityManager has been called by org.opensearch.bootstrap.OpenSearch (file:/usr/share/opensearch/lib/opensearch-2.18.0.jar)`,
		`WARNING: Using incubator modules: jdk.incubator.vector`,
	}

	for _, message := range cases {
		if severity, ok := classifyContainerLogIssue(message); ok {
			t.Fatalf("expected message to be ignored, got severity=%q for %q", severity, message)
		}
	}
}

func TestClassifyContainerLogIssue_StillFlagsRealIssues(t *testing.T) {
	cases := []struct {
		message  string
		severity string
	}{
		{
			message:  `[error] connect() failed (111: Connection refused) while connecting to upstream`,
			severity: "error",
		},
		{
			message:  `WARN: upstream cache is warming slowly`,
			severity: "warning",
		},
		{
			message:  `java.lang.IllegalStateException: shard lock failure`,
			severity: "error",
		},
	}

	for _, tc := range cases {
		severity, ok := classifyContainerLogIssue(tc.message)
		if !ok {
			t.Fatalf("expected message to be classified: %q", tc.message)
		}
		if severity != tc.severity {
			t.Fatalf("expected severity %q, got %q for %q", tc.severity, severity, tc.message)
		}
	}
}
