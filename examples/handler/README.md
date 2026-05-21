# Handler backend examples

Runnable YAML for all **`backend.handler`** types:

| Path prefix | `handler.type` | Notes |
|-------------|----------------|-------|
| `/static/text`, `/static/json` | `static_response` | `status_code`, `headers`, `body` |
| `/static/files` | `file_server` | Serves files from `./static` |
| `/static/templates` | `templates` | Go `html/template` with `{Path, Method}` |
| `/static/script/js` | `script` + `javascript` | goja async script, `ctx.fetch` available |
| `/static/script/go` | `script` + `go` | yaegi; `ctx` is `*zoox.Context` |

Host **`status.example.work`** uses a **rule-level** handler (no `paths`).

## Validate

From the repository root:

```bash
ingress validate -c examples/handler/ingress.yaml
```

## Run

`root_dir` paths are relative to the process working directory. Start from this directory:

```bash
cd examples/handler
ingress run -c ingress.yaml
```

## Try

```bash
curl -H "Host: handler.example.work" http://127.0.0.1:8080/static/text
curl -H "Host: handler.example.work" http://127.0.0.1:8080/static/json
curl -H "Host: handler.example.work" http://127.0.0.1:8080/static/files/hello.txt
curl -H "Host: handler.example.work" http://127.0.0.1:8080/static/templates/page.html
curl -H "Host: handler.example.work" http://127.0.0.1:8080/static/script/js
curl -H "Host: handler.example.work" http://127.0.0.1:8080/static/script/go
curl -H "Host: status.example.work" http://127.0.0.1:8080/
```

See also [`docs/guide/routing.md`](../../docs/guide/routing.md) (Handler Backend) and [`examples/advanced/service-mode-external-mixed.yaml`](../advanced/service-mode-external-mixed.yaml) for mixed service + handler paths.
