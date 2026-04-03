FROM golang:1.24-bookworm AS builder

WORKDIR /src
COPY runtime/config/launcher /src/runtime/config/launcher
COPY runtime/config/l4guard /src/runtime/config/l4guard

RUN CGO_ENABLED=0 GO111MODULE=off go build -o /out/waf-runtime-launcher ./runtime/config/launcher
RUN CGO_ENABLED=0 GO111MODULE=off go build -o /out/waf-runtime-l4-guard ./runtime/config/l4guard

FROM debian:12-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        iproute2 \
        iptables \
        nginx \
        libnginx-mod-http-modsecurity \
        modsecurity-crs \
        wget \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /out/waf-runtime-launcher /usr/local/bin/waf-runtime-launcher
COPY --from=builder /out/waf-runtime-l4-guard /usr/local/bin/waf-runtime-l4-guard
COPY runtime/image/entrypoint.sh /usr/local/bin/waf-runtime-entrypoint

RUN chmod +x /usr/local/bin/waf-runtime-launcher /usr/local/bin/waf-runtime-l4-guard /usr/local/bin/waf-runtime-entrypoint \
    && mkdir -p /var/lib/waf/active /var/lib/waf/candidates /etc/waf/nginx /etc/waf/modsecurity /etc/waf/tls /etc/waf/errors \
    && rm -f /etc/nginx/sites-enabled/default /etc/nginx/conf.d/default.conf

ENV WAF_RUNTIME_ROOT=/var/lib/waf

ENTRYPOINT ["/usr/local/bin/waf-runtime-entrypoint"]
