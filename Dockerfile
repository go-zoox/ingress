# Builder
FROM --platform=$BUILDPLATFORM whatwewant/builder-go:v1.22-1 as builder

WORKDIR /build

COPY go.mod ./

COPY go.sum ./

RUN go mod download

COPY . .

ARG TARGETARCH

RUN CGO_ENABLED=0 \
  GOOS=linux \
  GOARCH=$TARGETARCH \
  go build \
  -trimpath \
  -ldflags '-w -s -buildid=' \
  -v -o ingress ./cmd/ingress

# Server
FROM whatwewant/alpine:v3.17-1

LABEL MAINTAINER="Zero<tobewhatwewant@gmail.com>"

LABEL org.opencontainers.image.source="https://github.com/go-zoox/ingress"

ARG VERSION=latest

ENV TERMINAL_VERSION=${VERSION}

COPY --from=builder /build/ingress /bin

RUN ingress --version

CMD ingress run -c /etc/ingress/config.yaml
