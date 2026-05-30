# Admin authentication & RBAC

Minimal Admin Console login with **local Basic auth** (opt-in; default auth type is **`none`**).

Source: [`examples/admin-auth/`](https://github.com/go-zoox/ingress/tree/master/examples/admin-auth).

## Open mode (default)

When **`admin.auth.type`** is omitted, ingress uses **`none`** — no login page. Example:

<<< @/../examples/admin-auth/open-no-auth.yaml{yaml}

Use **`none`** only on localhost or trusted networks. For production, set **`basic`** or **`oauth`**.

## Basic login (recommended for production)

<<< @/../examples/admin-auth/ingress.yaml{yaml}

| Item | Value |
|------|--------|
| Admin URL | `http://127.0.0.1:9080` |
| Default login | `admin` / `admin` (from `admin.auth.basic`) |
| RBAC database | `./admin-auth.db` next to this YAML |

## Validate and run

```bash
ingress validate -c examples/admin-auth/ingress.yaml
ingress run -c examples/admin-auth/ingress.yaml
```

After login, open **权限** in the sidebar to manage users, roles, and permissions.

## Related docs

- [Admin console guide · Authentication & RBAC](/guide/admin#authentication--rbac)
- Full demo bundle with sample logs and WAF: [Admin console example](/examples/admin-console)
