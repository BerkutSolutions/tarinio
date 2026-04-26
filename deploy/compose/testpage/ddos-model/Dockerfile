FROM golang:1.26.2-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum ./
COPY cmd/tarinio-sentinel ./cmd/tarinio-sentinel
COPY internal/sentinel ./internal/sentinel
RUN CGO_ENABLED=0 go build -o /out/tarinio-sentinel ./cmd/tarinio-sentinel

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /out/tarinio-sentinel /usr/local/bin/tarinio-sentinel
ENTRYPOINT ["/usr/local/bin/tarinio-sentinel"]
