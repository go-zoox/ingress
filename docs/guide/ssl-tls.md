# SSL/TLS Configuration

Ingress supports SSL/TLS termination, allowing you to serve HTTPS traffic and terminate TLS connections at the proxy level.

## HTTPS Configuration

To enable HTTPS, configure the `https` section in your configuration file:

```yaml
version: v1
port: 8080

https:
  port: 8443
  ssl:
    - domain: example.com
      cert:
        certificate: /path/to/certificate.pem
        certificate_key: /path/to/private-key.pem
    - domain: api.example.com
      cert:
        certificate: /path/to/api-certificate.pem
        certificate_key: /path/to/api-private-key.pem
```

### Configuration Fields

| Field | Type | Description |
|-------|------|-------------|
| `port` | int | HTTPS port to listen on (default: 8443) |
| `ssl` | array | Array of SSL certificate configurations |

### SSL Certificate Configuration

Each SSL entry requires:

| Field | Type | Description |
|-------|------|-------------|
| `domain` | string | Domain name for the certificate |
| `cert.certificate` | string | Path to the certificate file (PEM format) |
| `cert.certificate_key` | string | Path to the private key file (PEM format) |

## Certificate Formats

Ingress expects certificates in PEM format. Both the certificate and private key should be in PEM format:

```
-----BEGIN CERTIFICATE-----
...
-----END CERTIFICATE-----
```

```
-----BEGIN PRIVATE KEY-----
...
-----END PRIVATE KEY-----
```

## Multiple Domains

You can configure multiple SSL certificates for different domains:

```yaml
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

## Certificate File Paths

Certificate files can be specified using:

- **Absolute paths**: `/etc/ssl/example.com/cert.pem`
- **Relative paths**: Relative to the working directory when Ingress starts

Make sure Ingress has read permissions for the certificate files.

## Using Let's Encrypt

You can use certificates from Let's Encrypt. Typically, Let's Encrypt certificates are stored in locations like:

```yaml
https:
  port: 8443
  ssl:
    - domain: example.com
      cert:
        certificate: /etc/letsencrypt/live/example.com/fullchain.pem
        certificate_key: /etc/letsencrypt/live/example.com/privkey.pem
```

## Certificate Reloading

When you update certificate files, you can reload the configuration without restarting:

```bash
# Send SIGHUP signal
kill -HUP $(cat /tmp/gozoox.ingress.pid)

# Or use the reload command
ingress reload
```

Ingress will reload the certificates from the configured paths.

## HTTP to HTTPS Redirect

To redirect HTTP traffic to HTTPS, you can configure a redirect rule:

```yaml
rules:
  - host: example.com
    backend:
      redirect:
        url: https://example.com
        permanent: true
```

Or handle it at the application level by checking the `X-Forwarded-Proto` header.

## SNI (Server Name Indication)

Ingress supports SNI, allowing it to serve different certificates for different domains on the same port. The certificate is selected based on the domain name in the TLS handshake.

## Backend Communication

When Ingress terminates TLS and forwards to backend services:

- Backend services can use HTTP (no TLS required)
- The original protocol information is preserved in headers like `X-Forwarded-Proto: https`
- Backend services can still use HTTPS if needed

Example configuration:

```yaml
https:
  port: 8443
  ssl:
    - domain: example.com
      cert:
        certificate: /path/to/cert.pem
        certificate_key: /path/to/key.pem

rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        protocol: http  # Backend uses HTTP, TLS terminated at Ingress
```

## Security Best Practices

1. **Use Strong Cipher Suites**: Ensure your certificates use strong encryption
2. **Keep Certificates Updated**: Regularly renew certificates before expiration
3. **Use Valid Certificates**: Avoid self-signed certificates in production
4. **Secure Private Keys**: Protect private key files with appropriate permissions (e.g., 600)
5. **TLS Version**: Use TLS 1.2 or higher
6. **Certificate Chain**: Include the full certificate chain in the certificate file
7. **Monitor Expiration**: Set up alerts for certificate expiration

## Troubleshooting

### Certificate Not Loading

- Verify certificate file paths are correct
- Check file permissions (Ingress needs read access)
- Ensure certificates are in PEM format
- Check certificate file syntax

### Certificate Mismatch

- Verify the domain in the certificate matches the request domain
- Check that SNI is working correctly
- Ensure the certificate hasn't expired

### Connection Refused

- Verify the HTTPS port is not already in use
- Check firewall rules allow traffic on the HTTPS port
- Ensure Ingress is listening on the correct port
