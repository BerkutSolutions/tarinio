FROM alpine:3.20

COPY deploy/compose/testpage/request-archive/entrypoint.sh /usr/local/bin/request-archive-entrypoint
RUN chmod +x /usr/local/bin/request-archive-entrypoint \
    && mkdir -p /archive \
    && chown 65532:4 /archive

USER 65532:4

ENTRYPOINT ["/usr/local/bin/request-archive-entrypoint"]
