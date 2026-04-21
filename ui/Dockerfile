FROM golang:1.25-alpine AS ui-tests

WORKDIR /src

COPY go.mod go.sum ./
COPY CHANGELOG.md README.md README.en.md ./
COPY docs ./docs
COPY ui/app ./ui/app
COPY ui/tests ./ui/tests

RUN go test ./ui/tests

FROM nginx:1.27-alpine

COPY ui/nginx.conf /etc/nginx/conf.d/default.conf
COPY ui/app /usr/share/nginx/html
