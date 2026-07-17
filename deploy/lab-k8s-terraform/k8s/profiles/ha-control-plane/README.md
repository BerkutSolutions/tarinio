# HA control-plane TLS prerequisite

Before applying this profile, create the TLS secret for the internal Redis
service. The certificate must have `redis` in its DNS SAN and be signed by the
included CA:

```sh
kubectl -n tarinio-lab create secret tls tarinio-lab-redis-tls \
  --cert=redis/tls.crt --key=redis/tls.key
kubectl -n tarinio-lab patch secret tarinio-lab-redis-tls \
  --type merge -p '{"data":{"ca.crt":"BASE64_ENCODED_CA"}}'
```

Use a protected internal CA. Do not replace this with a self-signed leaf
certificate or enable insecure TLS verification.
