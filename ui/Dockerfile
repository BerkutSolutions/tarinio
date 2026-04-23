FROM golang:1.25-alpine AS ui-tests

WORKDIR /src

RUN apk add --no-cache nodejs npm

COPY go.mod go.sum ./
COPY CHANGELOG.md README.md README.en.md ./
COPY package.json package-lock.json ./
COPY scripts/check-docs-lockfile.js ./scripts/check-docs-lockfile.js
COPY control-plane/internal/appmeta ./control-plane/internal/appmeta
COPY docs ./docs
COPY ui/app ./ui/app
COPY ui/nginx.conf ./ui/nginx.conf
COPY ui/tests ./ui/tests

RUN go test ./ui/tests

FROM nginx:1.27-alpine

COPY ui/nginx.conf /etc/nginx/conf.d/default.conf
COPY ui/app /usr/share/nginx/html
