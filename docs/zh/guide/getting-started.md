# 快速开始

Ingress 是一个强大、灵活的反向代理，可以轻松将流量路由到后端服务。本指南将帮助您开始使用 Ingress。

## 安装

### 使用 Go Install

安装 Ingress 最简单的方法是使用 `go install`：

```bash
go install github.com/go-zoox/ingress@latest
```

这会将 `ingress` 二进制文件安装到您的 `$GOPATH/bin` 目录（如果设置了 `$GOBIN`，则安装到该目录）。

### 使用 Docker

您也可以使用 Docker 运行 Ingress：

```bash
docker run -d \
  -p 8080:8080 \
  -v /path/to/ingress.yaml:/etc/ingress/config.yaml \
  gozoox/ingress:latest
```

### 从源码构建

如果您想从源码构建：

```bash
git clone https://github.com/go-zoox/ingress.git
cd ingress
go build -o ingress ./cmd/ingress
```

## 快速开始

### 1. 创建配置文件

创建一个名为 `ingress.yaml` 的文件：

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

### 2. 启动服务器

使用您的配置启动 Ingress：

```bash
ingress run -c ingress.yaml
```

或使用默认配置路径：

```bash
ingress run
```

如果未指定配置文件，默认配置路径为 `/etc/ingress/ingress.yaml`。

### 3. 测试设置

Ingress 运行后，您可以通过发送请求来测试：

```bash
curl -H "Host: example.com" http://localhost:8080
```

## 命令行选项

### Run 命令

```bash
ingress run [options]
```

选项：
- `-c, --config <path>`: 配置文件路径
- `-p, --port <port>`: 覆盖配置中的端口
- `--pid-file <path>`: PID 文件路径（默认：`/tmp/gozoox.ingress.pid`）

### Reload 命令

在不重启的情况下重新加载配置：

```bash
ingress reload
```

或向运行中的进程发送 SIGHUP 信号：

```bash
kill -HUP $(cat /tmp/gozoox.ingress.pid)
```

## 配置文件位置

Ingress 按以下顺序查找配置文件：

1. 由 `-c` 或 `--config` 标志指定的路径
2. 环境变量 `CONFIG`
3. 默认路径：`/etc/ingress/ingress.yaml`

## 下一步

- 了解[配置](/zh/guide/configuration)选项
- 探索[路由](/zh/guide/routing)功能
- 设置[认证](/zh/guide/authentication)
- 配置 [SSL/TLS](/zh/guide/ssl-tls)
