# Admin Console authentication & RBAC

Runnable configs for **Admin Console login** (distinct from route-level `backend.service.auth`).

| File | Purpose |
|------|---------|
| `open-no-auth.yaml` | **`admin.auth.type: none`** — default behavior when `auth` is omitted |
| `ingress.yaml` | **`admin.auth.type: basic`** — local login + RBAC seed (`admin` / `admin`) |

## Default: no login

Omitting **`admin.auth.type`** (or setting **`none`**) skips the login page. Validate and run the open example:

```bash
ingress validate -c examples/admin-auth/open-no-auth.yaml
ingress run -c examples/admin-auth/open-no-auth.yaml
```

Use **`none`** only on localhost or trusted networks.

## Basic login (production)

```bash
ingress validate -c examples/admin-auth/ingress.yaml
ingress run -c examples/admin-auth/ingress.yaml
```

Open **http://127.0.0.1:9080** and sign in with **`admin.auth.basic`** credentials.

On first start with **`basic`**, ingress seeds builtin roles and syncs the bootstrap super-admin user.

## Notes

- **`menu:*` permissions** control sidebar visibility; action grants alone do not show menu items.
- Relative **`admin.database.dsn`** paths resolve beside the ingress config file directory.
- Full demo bundle: [`examples/admin-console/`](../admin-console/) (explicit **`auth.type: basic`**).

Docs: [Admin console guide — Authentication & RBAC](../../docs/guide/admin.md#authentication--rbac).
