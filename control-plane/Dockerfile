FROM golang:1.25-bookworm AS builder

WORKDIR /src
COPY go.mod ./
COPY go.sum ./
COPY internal ./internal
COPY compiler ./compiler
COPY control-plane ./control-plane

RUN go build -o /out/waf-control-plane ./control-plane/cmd/control-plane
RUN go build -o /out/waf-cli ./control-plane/cmd/waf-cli

FROM docker:27-cli AS dockercli

FROM debian:12-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        bash \
        ca-certificates \
        nginx \
        libnginx-mod-http-modsecurity \
        wget \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /out/waf-control-plane /usr/local/bin/waf-control-plane
COPY --from=builder /out/waf-cli /usr/local/bin/waf-cli
COPY --from=builder /src/compiler/templates /src/compiler/templates
COPY scripts /src/scripts
COPY --from=dockercli /usr/local/bin/docker /usr/local/bin/docker

RUN mkdir -p /var/lib/waf/control-plane /var/lib/waf/active /var/lib/waf/candidates \
    && rm -f /etc/nginx/sites-enabled/default /etc/nginx/conf.d/default.conf

ENV CONTROL_PLANE_HTTP_ADDR=0.0.0.0:8080
ENV WAF_RUNTIME_ROOT=/var/lib/waf

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/waf-control-plane"]
