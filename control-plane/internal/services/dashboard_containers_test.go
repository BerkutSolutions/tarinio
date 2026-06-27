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
		`2026/04/24 14:31:20 [warn] tarinio-sentinel: emergency single-source flood detected: ip=203.0.113.40 rps=140`,
		`2026/04/24 14:31:20 [warn] tarinio-sentinel: emergency botnet burst detected: rps=240 unique_ips=240`,
		`2026-06-24 14:10:15.280 UTC [432] FATAL: the database system is shutting down`,
		`2026-06-24 14:10:15.154 UTC [233] FATAL: terminating connection due to administrator command`,
		`2026/06/24 14:08:58 [warn] 31#31: *261 an upstream response is buffered to a temporary file /var/cache/nginx/proxy_temp/1/00/0000000001 while reading upstream`,
		`2026/06/26 08:51:54 [warn] 66#66: *13 a client request body is buffered to a temporary file /var/lib/nginx/body/0000000001, client: 194.104.94.104, server: sentry.hantico.ru, request: "POST /api/2/envelope/?sentry_version=7&sentry_key=9b5b65904ce4c83885892449b76a465a&sentry_client=sentry.javascript.nuxt%2F10.40.0 HTTP/1.1", host: "sentry.hantico.ru", referrer: "https://tnm.hantico.ru/"`,
		`[2026-06-26T10:09:27,467][WARN ][o.o.w.QueryGroupTask     ] [4a5adaef9fbc] QueryGroup _id can't be null, It should be set before accessing it. This is abnormal behaviour`,
		`control-plane bootstrap: failed to connect to user=waf database=waf: dial tcp 172.18.0.8:5432: connect: connection refused`,
		`[2026-06-24T21:47:03,025][WARN ][o.o.m.j.JvmGcMonitorService] [7de753eb623f] [gc][13067] overhead, spent [836ms] collecting in the last [1.4s]`,
		`2026/06/24 18:09:04 [error] event service runtime security collector failed: Get "http://localhost:8081/security-events": dial tcp [::1]:8081: connect: connection refused`,
		`2026/06/27 01:53:12 [error] 69#69: *1395 directory index of "/var/lib/waf/control-plane/acme-challenges/" is forbidden, client: 195.178.110.199, server: n8n.hantico.com, request: "GET /.well-known/acme-challenge/ HTTP/1.1", host: "n8n.hantico.com"`,
		`2026/06/27 01:55:18 [error] 69#69: *1407 open() "/var/lib/waf/control-plane/acme-challenges/random-probe" failed (2: No such file or directory), client: 203.0.113.40, server: n8n.hantico.com, request: "GET /.well-known/acme-challenge/random-probe HTTP/1.1", host: "n8n.hantico.com"`,
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
