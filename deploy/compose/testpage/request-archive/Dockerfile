FROM alpine:3.20

RUN apk add --no-cache bash coreutils

COPY deploy/compose/testpage/request-archive/entrypoint.sh /usr/local/bin/request-archive-entrypoint
RUN chmod +x /usr/local/bin/request-archive-entrypoint

ENTRYPOINT ["/usr/local/bin/request-archive-entrypoint"]
