FROM golang:1.24-bookworm AS builder
WORKDIR /src
COPY deploy/compose/default/ddos-model/main.go /src/main.go
RUN CGO_ENABLED=0 GO111MODULE=off go build -o /out/ddos-model /src/main.go

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /out/ddos-model /usr/local/bin/ddos-model
ENTRYPOINT ["/usr/local/bin/ddos-model"]
