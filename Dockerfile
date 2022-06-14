# Builder
FROM golang:1.18-alpine as builder

RUN         apk add --no-cache make git

WORKDIR     /app

COPY        go.mod ./

COPY        go.sum ./

RUN         go mod download

ARG         VERSION=unknown

ARG         BUILD_TIME=unknown

ARG         COMMIT_HASH=unknown

COPY        . ./

RUN         CGO_ENABLED=0 \
            GOOS=linux \
            GOARCH=amd64 \
            go build \
              -trimpath \
              -ldflags '\
                -X "github.com/go-zoox/ingress/constants.Version=${VERSION}" \
                -X "github.com/go-zoox/ingress/constants.BuildTime=${BUILD_TIME}" \
                -X "github.com/go-zoox/ingress/constants.CommitHash=${COMMIT_HASH}" \
                -w -s -buildid= \
              ' \
              -v -o ingress

# Product
FROM  scratch

LABEL       MAINTAINER="Zero<tobewhatwewant@gmail.com>"

LABEL       org.opencontainers.image.source="https://github.com/go-zoox/ingress"

ARG         VERSION=v1.0.0

COPY        --from=builder /app/ingress /

COPY        conf/ingress.yaml /conf/ingress.yaml

EXPOSE      53

ENV         GIN_MODE=release

ENV         VERSION=${VERSION}

CMD  ["/ingress", "-c", "/conf/ingress.yaml"]
