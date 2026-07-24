FROM nginx:1.27-alpine

COPY deploy/compose/ha-lab/demo-app/nginx.conf /etc/nginx/conf.d/default.conf
COPY deploy/compose/ha-lab/demo-app/nginx.rootless.conf /etc/nginx/nginx.conf
COPY deploy/compose/ha-lab/demo-app/html /usr/share/nginx/html

RUN mkdir -p /var/cache/nginx /var/run \
    && chown -R 101:101 /var/cache/nginx /var/run

USER 101:101
