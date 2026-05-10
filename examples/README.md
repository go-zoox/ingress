# Example configurations

Runnable YAML samples live under this directory. Validate any file before deploy:

```bash
ingress validate -c examples/basic/ingress.yaml
```

| Directory | Topic |
|-----------|--------|
| `basic/` | Minimal host routing |
| `path-routing/` | Path-based backends |
| `authentication/` | Basic and bearer auth |
| `ssl-tls/` | HTTPS, certs, global redirect |
| `advanced/` | Regex hosts, rewrites, health, cache |
| `redirect/` | Backend redirects and capture templates |

The documentation site (`docs/examples/`) embeds these files via VitePress code snippets so examples stay in one place.
