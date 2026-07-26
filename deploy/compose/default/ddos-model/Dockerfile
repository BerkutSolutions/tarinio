FROM golang:1.26.5-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum ./
COPY cmd/tarinio-sentinel ./cmd/tarinio-sentinel
COPY internal/sentinel ./internal/sentinel
RUN CGO_ENABLED=0 go build -o /out/tarinio-sentinel ./cmd/tarinio-sentinel

FROM alpine:3.20
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /out/tarinio-sentinel /usr/local/bin/tarinio-sentinel
COPY deploy/compose/default/ddos-model/entrypoint.sh /usr/local/bin/tarinio-sentinel-entrypoint

RUN mkdir -p /state /out \
    && chown 65532:4 /state /out \
    && chmod 0755 /usr/local/bin/tarinio-sentinel-entrypoint \
    && printf 'tarinio:x:65532:4:tarinio sentinel:/nonexistent:/sbin/nologin\n' >> /etc/passwd

USER 65532:4

ENTRYPOINT ["/usr/local/bin/tarinio-sentinel-entrypoint"]
