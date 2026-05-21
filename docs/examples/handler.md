# Handler Backend Examples

Runnable samples for every **`backend.handler`** type. Source: [`examples/handler/`](https://github.com/go-zoox/ingress/tree/master/examples/handler).

Field reference and script APIs: [Routing guide — Handler Backend](/guide/routing#handler-backend).

## All handler types

<<< @/../examples/handler/ingress.yaml

Static assets for **`file_server`** and **`templates`** live beside the config:

- `examples/handler/static/` — plain files (`index.html`, `hello.txt`)
- `examples/handler/templates/` — Go templates with <span v-pre>{{.Path}}</span> and <span v-pre>{{.Method}}</span>

## Validate and run

```bash
ingress validate -c examples/handler/ingress.yaml
cd examples/handler && ingress run -c ingress.yaml
```

`handler.root_dir` is resolved relative to the **process working directory**, so run from `examples/handler/` (or adjust paths).

## Quick checks

```bash
curl -H "Host: handler.example.work" http://127.0.0.1:8080/static/text
curl -H "Host: handler.example.work" http://127.0.0.1:8080/static/files/hello.txt
curl -H "Host: handler.example.work" http://127.0.0.1:8080/static/script/js
curl -H "Host: status.example.work" http://127.0.0.1:8080/
```

## Related examples

- **`static_response` only** on mixed service routes: [`service-mode-external-mixed.yaml`](/examples/advanced#complex-path-rewriting)
- Handler response caching: [`http-response-cache.yaml`](/examples/advanced#http-response-cache-backend-cache)
