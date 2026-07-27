FROM golang:1.25-alpine AS ui-tests

WORKDIR /src

RUN apk add --no-cache nodejs npm python3

COPY go.mod go.sum ./
COPY CHANGELOG.md README.md README.en.md ./
COPY .gitlab-ci.yml ./
COPY package.json package-lock.json ./
COPY scripts ./scripts
COPY deploy/compose ./deploy/compose
COPY control-plane/Dockerfile ./control-plane/Dockerfile
COPY control-plane/internal/appmeta ./control-plane/internal/appmeta
COPY control-plane/apiroutes ./control-plane/apiroutes
COPY runtime/image/Dockerfile ./runtime/image/Dockerfile
COPY docs ./docs
COPY ui/app ./ui/app
COPY ui/nginx.conf ./ui/nginx.conf
COPY ui/nginx.testpage.conf ./ui/nginx.testpage.conf
COPY ui/Dockerfile ./ui/Dockerfile
COPY ui/tests ./ui/tests

RUN GOTOOLCHAIN=auto go test ./ui/tests

FROM nginx:1.27-alpine

COPY --chmod=0644 ui/nginx.conf /etc/nginx/conf.d/default.conf
COPY --chmod=0644 ui/nginx.rootless.conf /etc/nginx/nginx.conf
COPY ui/app /usr/share/nginx/html
RUN chmod -R a+rX /usr/share/nginx/html \
    && mkdir -p /var/cache/nginx /var/run \
    && chown -R 101:101 /var/cache/nginx /var/run

USER 101:101
