FROM golang:1.26.5-bookworm AS build
WORKDIR /src
COPY deploy/compose/e2e/dast-canary/main.go ./main.go
RUN CGO_ENABLED=0 go build -o /out/dast-canary ./main.go

FROM scratch
COPY --from=build /out/dast-canary /dast-canary
EXPOSE 8080
USER 65532:65532
ENTRYPOINT ["/dast-canary"]
