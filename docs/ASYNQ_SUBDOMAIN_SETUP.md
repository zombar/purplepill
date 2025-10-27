# Asynq Monitoring UI - Subdomain Setup

This document explains how to configure access to the Asynq monitoring UI via subdomain instead of a path-based proxy.

## Why Subdomain Instead of Path?

Asynqmon doesn't support running on a subpath (like `/asynq`). It must run at the root path `/`. Using a subdomain provides:
- ✅ Clean separation of concerns
- ✅ No URL rewriting complexity
- ✅ No API routing conflicts
- ✅ Native asynqmon functionality

## Architecture

```
External Request → Reverse Proxy → Docker Container
asynq.docutab.honker → Caddy/Nginx → docutab-asynqmon:8080
```

## Configuration

### 1. Docker Compose (Already Configured)

The asynqmon service in `docker-compose.staging.yml` is already set up:
```yaml
asynqmon:
  image: hibiken/asynqmon:latest
  container_name: docutab-asynqmon
  ports:
    - "9084:8080"  # Exposed for reverse proxy
  networks:
    - docutab-network  # Internal communication
    - proxy            # External access
```

### 2. DNS Configuration

Add an A record or CNAME for your asynq subdomain:

```
Type: A or CNAME
Host: asynq
Points to: <your-server-ip> or docutab.honker
TTL: 300 (or your preference)
```

Result: `asynq.docutab.honker` → `<server-ip>`

### 3. Reverse Proxy Configuration

#### Option A: Caddy (Recommended)

Add to your Caddyfile:

```caddy
asynq.docutab.honker {
    reverse_proxy docutab-asynqmon:8080

    # Optional: Enable compression
    encode gzip

    # Optional: Access logging
    log {
        output file /var/log/caddy/asynq.log
    }
}
```

Then reload Caddy:
```bash
docker exec caddy caddy reload --config /etc/caddy/Caddyfile
```

#### Option B: Nginx

Add to your nginx configuration:

```nginx
server {
    listen 443 ssl http2;
    server_name asynq.docutab.honker;

    # SSL configuration (managed by certbot or your SSL provider)
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://localhost:9084;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_cache_bypass $http_upgrade;
    }
}
```

Then reload nginx:
```bash
nginx -t && nginx -s reload
```

#### Option C: Traefik

Add labels to the asynqmon service in `docker-compose.staging.yml`:

```yaml
asynqmon:
  # ... existing config ...
  labels:
    - "traefik.enable=true"
    - "traefik.http.routers.asynqmon.rule=Host(`asynq.docutab.honker`)"
    - "traefik.http.routers.asynqmon.entrypoints=websecure"
    - "traefik.http.routers.asynqmon.tls.certresolver=myresolver"
    - "traefik.http.services.asynqmon.loadbalancer.server.port=8080"
```

## Verification

After configuration, verify the setup:

1. **Check DNS resolution:**
   ```bash
   dig asynq.docutab.honker
   nslookup asynq.docutab.honker
   ```

2. **Test connection:**
   ```bash
   curl -I https://asynq.docutab.honker
   ```

3. **Access in browser:**
   ```
   https://asynq.docutab.honker
   ```

You should see the Asynqmon dashboard with:
- Queue statistics
- Active/scheduled/retry tasks
- Server information
- Metrics graphs (if Prometheus is configured)

## Troubleshooting

### 502 Bad Gateway
- Check if asynqmon container is running: `docker ps | grep asynqmon`
- Check container logs: `docker logs docutab-asynqmon`
- Verify proxy network: `docker network inspect proxy`

### SSL/TLS Errors
- Ensure your reverse proxy has valid certificates for `asynq.docutab.honker`
- For Caddy, it handles Let's Encrypt automatically
- For Nginx, run certbot: `certbot --nginx -d asynq.docutab.honker`

### Connection Refused
- Check if port 9084 is accessible from reverse proxy
- Verify firewall rules allow traffic
- Check if asynqmon is listening: `docker exec docutab-asynqmon netstat -ln | grep 8080`

### No Data Showing
- Verify Redis connection: Check `REDIS_ADDR` environment variable
- Check if workers are running and connected to the same Redis instance
- Review asynqmon logs for connection errors

## Security Considerations

1. **Authentication**: Consider adding authentication at the reverse proxy level:
   ```caddy
   asynq.docutab.honker {
       basicauth {
           admin $2a$14$...  # bcrypt hash
       }
       reverse_proxy docutab-asynqmon:8080
   }
   ```

2. **IP Whitelist**: Restrict access to specific IPs if needed

3. **Read-Only Mode**: Run asynqmon in read-only mode to prevent accidental queue modifications:
   ```yaml
   command: ["--enable-metrics-exporter", "--prometheus-addr=http://prometheus:9090", "--read-only"]
   ```

## Migration from Path-Based Proxy

If you were previously accessing asynqmon at `https://docutab.honker/asynq`:

1. Update any bookmarks to use `https://asynq.docutab.honker`
2. Update documentation/wikis with the new URL
3. The old path will return 404 after deploying the new nginx configuration
