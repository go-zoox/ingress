# Admin web UI (embedded into Go binary via core/admin/static)
FROM --platform=$BUILDPLATFORM node:22-alpine AS web

WORKDIR /build/core/admin/web

RUN corepack enable && corepack prepare pnpm@10.15.0 --activate

COPY core/admin/web/package.json core/admin/web/pnpm-lock.yaml ./

RUN pnpm install --frozen-lockfile

COPY core/admin/web/ ./

RUN pnpm build

# Builder
FROM --platform=$BUILDPLATFORM whatwewant/builder-go:v1.25-1 as builder

WORKDIR /build

COPY go.mod ./

COPY go.sum ./

RUN go mod download

COPY . .

COPY --from=web /build/core/admin/static/dist ./core/admin/static/dist

ARG TARGETARCH

RUN CGO_ENABLED=0 \
  GOOS=linux \
  GOARCH=$TARGETARCH \
  go build \
  -tags adminui \
  -trimpath \
  -ldflags '-w -s -buildid=' \
  -v -o ingress ./cmd/ingress

# Server
FROM whatwewant/alpine:v3-1

LABEL MAINTAINER="Zero<tobewhatwewant@gmail.com>"

LABEL org.opencontainers.image.source="https://github.com/go-zoox/ingress"

RUN mkdir -p /etc/geoip \
  && apk add --no-cache ca-certificates wget \
  && wget -qO /etc/geoip/GeoLite2-City.mmdb \
    https://github.com/go-zoox/geoip/releases/download/v0.0.3/GeoLite2-City.mmdb \
  && apk del wget

COPY --from=builder /build/ingress /bin

CMD ingress run -c /etc/ingress/config.yaml
