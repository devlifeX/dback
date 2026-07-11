# Reverse proxy and TLS

DBack listens on HTTP by default (`127.0.0.1:14127`). Terminate TLS at a reverse proxy for production.

## Caddy

```caddy
dback.example.com {
    encode gzip

    reverse_proxy 127.0.0.1:14127 {
        header_up X-Forwarded-For {remote_host}
        header_up X-Forwarded-Proto {scheme}
    }
}
```

Restrict `/metrics` to internal networks:

```caddy
@metrics path /metrics
handle @metrics {
    @internal remote_ip 10.0.0.0/8 172.16.0.0/12 192.168.0.0/16
    handle @internal {
        reverse_proxy 127.0.0.1:14127
    }
    respond 403
}
```

## nginx

```nginx
server {
    listen 443 ssl http2;
    server_name dback.example.com;

    ssl_certificate     /etc/ssl/dback/fullchain.pem;
    ssl_certificate_key /etc/ssl/dback/privkey.pem;

    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Content-Type-Options nosniff always;
    add_header X-Frame-Options DENY always;

    location /metrics {
        allow 10.0.0.0/8;
        allow 192.168.0.0/16;
        deny all;
        proxy_pass http://127.0.0.1:14127;
    }

    location / {
        proxy_pass http://127.0.0.1:14127;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # SSE operations stream
        proxy_buffering off;
        proxy_read_timeout 3600s;
    }
}
```

## Security notes

- Keep `DBACK_LISTEN` on loopback when using a reverse proxy.
- Do not expose `/metrics` publicly without ACLs.
- API token is required for all `/api/v1/*` routes except `version` and `openapi.json`.
- Web UI stores the bearer token in memory only; prefer HTTPS for token entry.

## Rate limiting

Application-level per-IP rate limiting is enabled on authenticated API routes (`DBACK_RATE_LIMIT_RPS`). Tune at the proxy for additional protection against abuse.
