# Handler 后端示例

覆盖全部 **`backend.handler`** 类型的可运行示例。源码：[`examples/handler/`](https://github.com/go-zoox/ingress/tree/master/examples/handler)。

字段说明与脚本 API 见 [路由指南 — Handler 后端](/zh/guide/routing#handler-后端)。

## 全部 handler 类型

<<< @/../examples/handler/ingress.yaml

**`file_server`** / **`templates`** 使用的静态文件与模板目录：

- `examples/handler/static/` — 普通文件（`index.html`、`hello.txt`）
- `examples/handler/templates/` — Go 模板，可用 <span v-pre>{{.Path}}</span>、<span v-pre>{{.Method}}</span>

## 校验与运行

```bash
ingress validate -c examples/handler/ingress.yaml
cd examples/handler && ingress run -c ingress.yaml
```

`handler.root_dir` 相对 **进程工作目录** 解析，因此建议在 `examples/handler/` 下启动（或自行调整路径）。

## 快速验证

```bash
curl -H "Host: handler.example.work" http://127.0.0.1:8080/static/text
curl -H "Host: handler.example.work" http://127.0.0.1:8080/static/files/hello.txt
curl -H "Host: handler.example.work" http://127.0.0.1:8080/static/script/js
curl -H "Host: status.example.work" http://127.0.0.1:8080/
```

## 相关示例

- 与 **service** 混用的 **`static_response`** 路径：[`service-mode-external-mixed.yaml`](/zh/examples/advanced#复杂路径重写)
- Handler 响应缓存：[`http-response-cache.yaml`](/zh/examples/advanced#http-响应缓存-backend-cache)
