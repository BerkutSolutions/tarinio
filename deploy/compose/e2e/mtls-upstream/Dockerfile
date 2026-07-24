FROM golang:1.26.5-bookworm AS build
WORKDIR /src
COPY deploy/compose/e2e/mtls-upstream/main.go ./main.go
RUN CGO_ENABLED=0 go build -o /out/mtls-upstream ./main.go
FROM scratch
COPY --from=build /out/mtls-upstream /mtls-upstream
EXPOSE 8080 8443
USER 65532:65532
ENTRYPOINT ["/mtls-upstream"]
