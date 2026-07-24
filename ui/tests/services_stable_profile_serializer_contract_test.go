package tests

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestServicesStableProfileSerializerKeepsAdvancedSettings(t *testing.T) {
	script := `
import { defaultSiteDraft, draftToEasyProfile } from "./ui/app/static/js/pages/sites.draft-core.js";
const draft = defaultSiteDraft();
Object.assign(draft, {
  id: "serializer-contract",
  primary_host: "serializer-contract.test",
  health_check_enabled: true,
  health_check_path: "/ready",
  health_check_interval_seconds: 17,
  health_check_fail_threshold: 4,
  use_ws_inspection: true,
  ws_block_patterns: ["(?i)blocked"],
  ws_max_message_bytes: 4096,
  ws_rate_msg_per_sec: 12,
  use_limit_req: true,
  limit_req_rate: "7r/m",
  mtls_enabled: true,
  mtls_optional: true,
  mtls_verify_depth: 3,
  mtls_client_ca_ref: "front-ca",
  mtls_pass_headers: true,
  upstream_mtls_enabled: true,
  upstream_mtls_cert_ref: "upstream-cert",
  upstream_mtls_key_ref: "upstream-key",
  upstream_mtls_ca_ref: "upstream-ca",
  auth_mode: "basic_or_token",
  auth_order: "antibot_first",
  auth_exclusion_rules: [{ path: "/public", methods: ["GET"] }],
  auth_service_tokens: [{ service_name: "probe", token: "secret", enabled: true }],
});
const profile = draftToEasyProfile(draft);
const expected = {
  health: profile.upstream_routing.health_check_enabled === true && profile.upstream_routing.health_check_path === "/ready",
  websocket: profile.security_websocket.use_ws_inspection === true && profile.security_websocket.ws_block_patterns.includes("(?i)blocked"),
  rateUnit: profile.security_behavior_and_limits.use_limit_req === true && profile.security_behavior_and_limits.limit_req_rate === "7r/m",
  frontMTLS: profile.front_service.mtls_enabled === true && profile.front_service.mtls_client_ca_ref === "front-ca",
  upstreamMTLS: profile.upstream_routing.upstream_mtls_enabled === true && profile.upstream_routing.upstream_mtls_cert_ref === "upstream-cert",
  auth: profile.security_auth_basic.auth_mode === "basic_or_token" && profile.security_auth_basic.auth_order === "antibot_first" && profile.security_auth_basic.exclusion_rules.length === 1 && profile.security_auth_basic.service_tokens.length === 1,
};
for (const [name, passed] of Object.entries(expected)) {
  if (!passed) throw new Error(name + " missing from stable profile payload: " + JSON.stringify(profile));
}
`
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	cmd := nodeESMCommand(t, script)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("stable profile serializer contract failed: %v: %s", err, strings.TrimSpace(string(output)))
	}
}
