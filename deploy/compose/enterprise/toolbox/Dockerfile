FROM golang:1.25-bookworm AS builder

WORKDIR /src
COPY go.mod ./
COPY go.sum ./
COPY internal ./internal
COPY control-plane ./control-plane

RUN go build -o /out/waf-cli ./control-plane/cmd/waf-cli

FROM debian:12-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        jq \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /out/waf-cli /usr/local/bin/waf-cli
COPY deploy/compose/ha-lab/tools /tools

RUN chmod +x /tools/*.sh

ENTRYPOINT ["/bin/sh", "-lc"]
CMD ["sleep infinity"]
