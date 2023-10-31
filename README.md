# Ingress - A Easy, Powerful, Fexible Reverse Proxy

[![PkgGoDev](https://pkg.go.dev/badge/github.com/go-zoox/ingress)](https://pkg.go.dev/github.com/go-zoox/ingress)
[![Build Status](https://github.com/go-zoox/ingress/actions/workflows/release.yml/badge.svg?branch=master)](https://github.com/go-zoox/ingress/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/go-zoox/ingress)](https://goreportcard.com/report/github.com/go-zoox/ingress)
[![Coverage Status](https://coveralls.io/repos/github/go-zoox/ingress/badge.svg?branch=master)](https://coveralls.io/github/go-zoox/ingress?branch=master)
[![GitHub issues](https://img.shields.io/github/issues/go-zoox/ingress.svg)](https://github.com/go-zoox/ingress/issues)
[![Release](https://img.shields.io/github/tag/go-zoox/ingress.svg?label=Release)](https://github.com/go-zoox/ingress/tags)


## Installation
To install the package, run:
```bash
go install github.com/go-zoox/ingress@latest
```

## Quick Start

```bash
# start ingress, cached in memory, default udp port: 80
sudo ingress

# start ingress with config (see conf/ingress.yml for more options)
sudo ingress -c ingress.yml
```

## Configuration
See the [configuration file](conf/ingress.yml).

## License
GoZoox is released under the [MIT License](./LICENSE).
