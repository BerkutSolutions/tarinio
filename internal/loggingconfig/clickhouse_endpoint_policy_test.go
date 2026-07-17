package loggingconfig

import "testing"

func TestValidateClickHouseEndpointRequiresExactDeploymentAllowlist(t *testing.T) {
	allowed := []string{"https://clickhouse.internal:8443", "http://clickhouse:8123"}
	if got, err := ValidateClickHouseEndpoint("http://clickhouse:8123/", allowed); err != nil || got != "http://clickhouse:8123" {
		t.Fatalf("expected configured endpoint, got %q, %v", got, err)
	}
	for _, endpoint := range []string{
		"http://169.254.169.254/latest/meta-data",
		"http://attacker.example:8123",
		"http://user:password@clickhouse:8123",
		"ftp://clickhouse:8123",
	} {
		if _, err := ValidateClickHouseEndpoint(endpoint, allowed); err == nil {
			t.Fatalf("expected endpoint %q to be rejected", endpoint)
		}
	}
}
