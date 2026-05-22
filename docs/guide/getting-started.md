# Getting Started

Ingress is a powerful, flexible reverse proxy that makes it easy to route traffic to your backend services. This guide will help you get started with Ingress.

## Installation

### Using Go Install

The easiest way to install Ingress is using `go install`:

```bash
go install github.com/go-zoox/ingress@latest
```

This will install the `ingress` binary to your `$GOPATH/bin` directory (or `$GOBIN` if set).

### Using Docker

You can also run Ingress using Docker:

```bash
docker run -d \
  -p 8080:8080 \
  -v /path/to/ingress.yaml:/etc/ingress/config.yaml \
  gozoox/ingress:latest
```

### Building from Source

If you want to build from source:

```bash
git clone https://github.com/go-zoox/ingress.git
cd ingress
go build -o ingress ./cmd/ingress
```

## Quick Start

### 1. Create a Configuration File

Create a file named `ingress.yaml`:

```yaml
version: v1
port: 8080

rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
```

Same routing outcome if you spell **`backend.type`** explicitly (see **`examples/basic/ingress.yaml`** for both hosts in one file):

```yaml
rules:
  - host: example.com
    backend:
      type: service
      service:
        name: backend-service
        port: 8080
```

Ingress infers **`backend.type`** when only **`backend.service`**, **`backend.handler`**, or **`backend.redirect`** applies—you normally **omit `type`**. Set **`backend.type` explicitly** only if **`ingress validate`** reports an ambiguous backend. See the [Routing guide](/guide/routing).

### 2. Start the Server

Start Ingress with your configuration:

```bash
ingress run -c ingress.yaml
```

Or use the default configuration path:

```bash
ingress run
```

The default configuration path is `/etc/ingress/config.yaml` if no config file is specified.

### 3. Test the Setup

Once Ingress is running, you can test it by making a request:

```bash
curl -H "Host: example.com" http://localhost:8080
```

## Proxying external HTTPS origins

For third-party or off-cluster HTTPS upstreams, set **`backend.service.mode: external`** so the origin receives its own hostname in **`Host`** (see [Rewriting](/guide/rewriting)). Legacy **`backend.mode`** works when it matches.

```yaml
rules:
  - host: mirror.example.com
    backend:
      service:
        mode: external
        protocol: https
        name: upstream.example.org
```

## Validate before run

Check YAML syntax, router compilation, TLS shape, and backend consistency without starting the server:

```bash
ingress validate -c ingress.yaml
```

`ingress run` and **`ingress reload`** perform the same validation; failures block startup or reload.

## Admin console (optional)

Enable the embedded operations UI and API in the same process:

```yaml
admin:
  enabled: true
  port: 9080
```

```bash
ingress run -c examples/admin-console/ingress.yaml
```

See the [Admin console guide](/guide/admin) and [admin-console example](/examples/admin-console).

## Command Line Options

Ingress exposes three subcommands: **`run`**, **`validate`**, and **`reload`**.

### Run Command

```bash
ingress run [options]
```

Options:
- `-c, --config <path>`: Path to the configuration file
- `-p, --port <port>`: Override the port from configuration
- `--pid-file <path>`: Path to the PID file (default: `/tmp/gozoox.ingress.pid`)

When **`admin.enabled: true`**, the admin server listens on **`admin.port`** (default **9080**) in the same process.

### Validate Command

```bash
ingress validate -c ingress.yaml
```

Uses the same config path resolution as **`run`** (`-c`, `CONFIG`, or `/etc/ingress/config.yaml`).

### Reload Command

Reload the configuration without restarting (validates first):

```bash
ingress reload -c ingress.yaml
```

Or send a SIGHUP signal to the running process:

```bash
kill -HUP $(cat /tmp/gozoox.ingress.pid)
```

The admin console **`POST /api/v1/reload`** and **Publish** flow trigger the same in-process reload when started via **`ingress run`**.

## Configuration File Location

Ingress looks for configuration files in the following order:

1. Path specified by `-c` or `--config` flag
2. Environment variable `CONFIG`
3. Default path: `/etc/ingress/config.yaml`

## Next Steps

- Learn about [Configuration](/guide/configuration) options
- Explore [Routing](/guide/routing) capabilities
- Optional [Admin console](/guide/admin) for logs, routes, and config publish
- [Request and Response Rewriting](/guide/rewriting): **`service.mode`** and **`Host`** defaults
- Set up [Authentication](/guide/authentication)
- Configure [SSL/TLS](/guide/ssl-tls)
