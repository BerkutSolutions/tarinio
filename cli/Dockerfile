FROM golang:1.26.2-bookworm AS builder

WORKDIR /src
COPY go.mod ./
COPY control-plane ./control-plane

RUN go build -o /out/waf-cli ./control-plane/cmd/waf-cli

FROM debian:12-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /out/waf-cli /usr/local/bin/waf-cli

ENTRYPOINT ["/usr/local/bin/waf-cli"]
