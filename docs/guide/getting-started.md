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

### 2. Start the Server

Start Ingress with your configuration:

```bash
ingress run -c ingress.yaml
```

Or use the default configuration path:

```bash
ingress run
```

The default configuration path is `/etc/ingress/ingress.yaml` if no config file is specified.

### 3. Test the Setup

Once Ingress is running, you can test it by making a request:

```bash
curl -H "Host: example.com" http://localhost:8080
```

## Command Line Options

### Run Command

```bash
ingress run [options]
```

Options:
- `-c, --config <path>`: Path to the configuration file
- `-p, --port <port>`: Override the port from configuration
- `--pid-file <path>`: Path to the PID file (default: `/tmp/gozoox.ingress.pid`)

### Reload Command

Reload the configuration without restarting:

```bash
ingress reload
```

Or send a SIGHUP signal to the running process:

```bash
kill -HUP $(cat /tmp/gozoox.ingress.pid)
```

## Configuration File Location

Ingress looks for configuration files in the following order:

1. Path specified by `-c` or `--config` flag
2. Environment variable `CONFIG`
3. Default path: `/etc/ingress/ingress.yaml`

## Next Steps

- Learn about [Configuration](/guide/configuration) options
- Explore [Routing](/guide/routing) capabilities
- Set up [Authentication](/guide/authentication)
- Configure [SSL/TLS](/guide/ssl-tls)
