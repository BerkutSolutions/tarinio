# Secure OpenSearch and ClickHouse profile

This profile never creates or rotates storage credentials or TLS keys. Before
applying it, create these immutable, externally managed secrets in
`tarinio-lab`:

- `tarinio-lab-opensearch-tls`: `node.crt`, `tls.key`, `ca.crt`. The node
  certificate must have subject `CN=tarinio-opensearch` and SAN `opensearch`.
- `tarinio-lab-opensearch-security`: OpenSearch Security YAML files, including
  a `waf-runtime` user whose password hash matches
  `tarinio-lab-secrets/OPENSEARCH_PASSWORD`. Its admin certificate DN is
  `CN=tarinio-opensearch-admin`.
- `tarinio-lab-clickhouse-tls`: `tls.crt`, `tls.key`, `ca.crt`, with `clickhouse`
  in the server certificate SANs.
- `tarinio-lab-clickhouse-users`: `50-waf-runtime.xml`, defining only the
  non-default `waf-runtime` user with a password hash matching
  `tarinio-lab-secrets/CLICKHOUSE_PASSWORD` and the required `waf_logs`
  grants.

Both stores receive only read-only secret mounts. OpenSearch security is
enabled with TLS on port 9200; ClickHouse disables plaintext HTTP/native ports
and exposes HTTPS only on 8443. The profile's NetworkPolicies allow those
ports only from control-plane and runtime pods.

For an upgrade, preserve the existing secret objects and passwords, then apply
the profile. Do not replace a key merely to enable the profile: first issue a
second trusted certificate and roll workloads only after both the old and new
CA chain are trusted.

After rollout, run:

```powershell
powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/k8s/scripts/verify-secure-storage-profile.ps1
```

The verifier is read-only: it checks secret presence, rollout state, TLS-only
ports, OpenSearch plugin state and the two NetworkPolicies without reading,
logging, rotating or replacing credentials.
