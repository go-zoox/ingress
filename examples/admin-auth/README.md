# Admin Console authentication & RBAC

Runnable configs for **Admin Console login** (distinct from route-level `backend.service.auth`).

| File | Purpose |
|------|---------|
| `ingress.yaml` | **`admin.auth.type: basic`** — local username/password login backed by RBAC (default **`admin` / `admin`**) |
| `open-no-auth.yaml` | **`admin.auth.type: none`** — open admin API/UI (local dev only) |

## Validate and run

```bash
ingress validate -c examples/admin-auth/ingress.yaml
ingress run -c examples/admin-auth/ingress.yaml
```

Open **http://127.0.0.1:9080** and sign in with the credentials under `admin.auth.basic`.

On first start, ingress seeds:

- Builtin permissions (action grants + **`menu:*`** sidebar visibility)
- Five builtin roles: **`admin`**, **`viewer`**, **`operator`**, **`developer`**, **`security`**
- RBAC user matching **`admin.auth.basic.username`** with the **`admin`** (super-admin) role

Manage users, roles, and permissions in the UI under **权限** after login.

## Notes

- **`menu:*` permissions** control sidebar visibility; action grants such as `routes:read` alone do not show a menu item.
- Login is rejected when the account has **no visible menus** (HTTP 403), even if the password is correct.
- Relative **`admin.database.dsn`** paths resolve beside the ingress config file directory.
- For the full demo bundle (sample logs, WAF, TLS), see [`examples/admin-console/`](../admin-console/).

Docs: [Admin console guide — Authentication & RBAC](../../docs/guide/admin.md#authentication--rbac).
