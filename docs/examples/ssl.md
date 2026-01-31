# SSL/TLS Examples

This page provides examples of SSL/TLS configuration.

## Basic HTTPS Configuration

```yaml
version: v1
port: 8080

https:
  port: 8443
  ssl:
    - domain: example.com
      cert:
        certificate: /etc/ssl/example.com/fullchain.pem
        certificate_key: /etc/ssl/example.com/privkey.pem
```

## Multiple Domains

```yaml
version: v1
port: 8080

https:
  port: 8443
  ssl:
    - domain: example.com
      cert:
        certificate: /etc/ssl/example.com/fullchain.pem
        certificate_key: /etc/ssl/example.com/privkey.pem
    - domain: api.example.com
      cert:
        certificate: /etc/ssl/api.example.com/fullchain.pem
        certificate_key: /etc/ssl/api.example.com/privkey.pem
    - domain: admin.example.com
      cert:
        certificate: /etc/ssl/admin.example.com/fullchain.pem
        certificate_key: /etc/ssl/admin.example.com/privkey.pem
```

## Let's Encrypt Certificates

```yaml
version: v1
port: 8080

https:
  port: 8443
  ssl:
    - domain: example.com
      cert:
        certificate: /etc/letsencrypt/live/example.com/fullchain.pem
        certificate_key: /etc/letsencrypt/live/example.com/privkey.pem
```

## HTTPS with Backend Services

```yaml
version: v1
port: 8080

https:
  port: 8443
  ssl:
    - domain: example.com
      cert:
        certificate: /etc/ssl/example.com/fullchain.pem
        certificate_key: /etc/ssl/example.com/privkey.pem

rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        protocol: http  # Backend uses HTTP, TLS terminated at Ingress
```

## HTTP to HTTPS Redirect

```yaml
version: v1
port: 8080

https:
  port: 8443
  ssl:
    - domain: example.com
      cert:
        certificate: /etc/ssl/example.com/fullchain.pem
        certificate_key: /etc/ssl/example.com/privkey.pem

rules:
  - host: example.com
    backend:
      redirect:
        url: https://example.com
        permanent: true
```

## Testing

### HTTPS Request

```bash
curl https://example.com:8443/api
```

### Verify Certificate

```bash
openssl s_client -connect example.com:8443 -servername example.com
```

### Certificate Reload

After updating certificates, reload the configuration:

```bash
kill -HUP $(cat /tmp/gozoox.ingress.pid)
```
