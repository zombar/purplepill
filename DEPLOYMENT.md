# Staging Deployment Guide

## Prerequisites

1. **External Reverse Proxy** configured to route:
   - `https://honker/` → `localhost:3001` (Web UI)
   - `https://honker/api/*` → `localhost:9080` (Controller API)
   - `https://honker/content/*` → `localhost:9080` (SEO content)

2. **Ollama** running and accessible:
   - Update `OLLAMA_URL` in `.env.staging` to match your setup
   - Required models: `gemma3:12b`

3. **Docker & Docker Compose** installed

## Quick Deploy

```bash
# 1. Copy environment file
cp .env.staging .env

# 2. Update Ollama URL in .env if needed
nano .env

# 3. Build images
docker-compose -f docker-compose.yml -f docker-compose.staging.yml build

# 4. Start services
docker-compose -f docker-compose.yml -f docker-compose.staging.yml up -d

# 5. Check logs
docker-compose -f docker-compose.yml -f docker-compose.staging.yml logs -f

# 6. Check health
curl http://localhost:9080/health
curl http://localhost:3001/
```

## Exposed Ports

| Port | Service | Purpose |
|------|---------|---------|
| 3001 | Web UI | Main web interface (reverse proxy this) |
| 9080 | Controller | API & SEO content (reverse proxy this) |
| 9081 | Scraper | Optional: debugging only |
| 9082 | TextAnalyzer | Optional: debugging only |
| 9083 | Scheduler | Optional: debugging only |

## Example Reverse Proxy Configs

### Caddy

```caddy
honker {
    reverse_proxy /api/* localhost:9080
    reverse_proxy /content/* localhost:9080
    reverse_proxy /* localhost:3001
}
```

### Nginx

#### Option 1: Basic Configuration

Save as `/etc/nginx/sites-available/purpletab.conf`:

```nginx
# Upstream definitions
upstream purpletab_web {
    server localhost:3001;
    keepalive 32;
}

upstream purpletab_api {
    server localhost:9080;
    keepalive 32;
}

# HTTP redirect to HTTPS
server {
    listen 80;
    server_name honker;

    location / {
        return 301 https://$server_name$request_uri;
    }
}

# Main HTTPS server
server {
    listen 443 ssl http2;
    server_name honker;

    # SSL Configuration
    ssl_certificate /etc/letsencrypt/live/honker/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/honker/privkey.pem;
    ssl_session_timeout 1d;
    ssl_session_cache shared:SSL:50m;
    ssl_session_tickets off;

    # Modern SSL configuration
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;

    # Security Headers
    add_header Strict-Transport-Security "max-age=63072000; includeSubDomains; preload" always;
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "no-referrer-when-downgrade" always;

    # Logging
    access_log /var/log/nginx/purpletab-access.log;
    error_log /var/log/nginx/purpletab-error.log;

    # Client limits
    client_max_body_size 100M;
    client_body_timeout 120s;

    # API routes
    location /api/ {
        proxy_pass http://purpletab_api;

        # Proxy headers
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-Host $host;
        proxy_set_header X-Forwarded-Port $server_port;

        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;

        # Connection reuse
        proxy_http_version 1.1;
        proxy_set_header Connection "";
    }

    # SEO content routes
    location /content/ {
        proxy_pass http://purpletab_api;

        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # Cache SEO content for better performance
        proxy_cache_bypass $http_pragma $http_authorization;
        proxy_no_cache $http_pragma $http_authorization;
        add_header X-Cache-Status $upstream_cache_status;

        proxy_http_version 1.1;
        proxy_set_header Connection "";
    }

    # Health check endpoint (optional, direct access)
    location /health {
        proxy_pass http://purpletab_api/health;
        access_log off;
        proxy_set_header Host $host;
    }

    # Web UI - catch all
    location / {
        proxy_pass http://purpletab_web;

        # Proxy headers
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket support (if needed for hot reload in dev)
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        # Connection reuse
        proxy_http_version 1.1;

        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }
}
```

Enable the site:
```bash
sudo ln -s /etc/nginx/sites-available/purpletab.conf /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

#### Option 2: With Rate Limiting

Add rate limiting zones to `/etc/nginx/nginx.conf` inside the `http` block:

```nginx
http {
    # ... other config ...

    # Rate limiting zones
    limit_req_zone $binary_remote_addr zone=api_limit:10m rate=10r/s;
    limit_req_zone $binary_remote_addr zone=content_limit:10m rate=20r/s;
    limit_req_zone $binary_remote_addr zone=general_limit:10m rate=100r/s;

    # Connection limiting
    limit_conn_zone $binary_remote_addr zone=addr:10m;

    # ... rest of config ...
}
```

Then modify `/etc/nginx/sites-available/purpletab.conf`:

```nginx
# Add to the server block locations:

    # API routes with rate limiting
    location /api/ {
        limit_req zone=api_limit burst=20 nodelay;
        limit_conn addr 10;

        proxy_pass http://purpletab_api;
        # ... rest of proxy config ...
    }

    # SEO content with rate limiting
    location /content/ {
        limit_req zone=content_limit burst=50 nodelay;

        proxy_pass http://purpletab_api;
        # ... rest of proxy config ...
    }

    # Web UI with rate limiting
    location / {
        limit_req zone=general_limit burst=200 nodelay;

        proxy_pass http://purpletab_web;
        # ... rest of proxy config ...
    }
```

#### Option 3: With Caching

Add to `/etc/nginx/nginx.conf` in the `http` block:

```nginx
http {
    # Cache configuration
    proxy_cache_path /var/cache/nginx/purpletab levels=1:2 keys_zone=purpletab_cache:10m max_size=1g inactive=60m use_temp_path=off;

    # ... rest of config ...
}
```

Then in your site config:

```nginx
    # Cached SEO content
    location /content/ {
        proxy_cache purpletab_cache;
        proxy_cache_valid 200 60m;
        proxy_cache_valid 404 10m;
        proxy_cache_use_stale error timeout updating http_500 http_502 http_503 http_504;
        proxy_cache_background_update on;
        proxy_cache_lock on;

        add_header X-Cache-Status $upstream_cache_status;

        proxy_pass http://purpletab_api;
        # ... rest of proxy config ...
    }
```

#### Option 4: Dockerized Nginx

If you prefer to run Nginx in Docker alongside PurpleTab, create `nginx.conf`:

```nginx
events {
    worker_connections 1024;
}

http {
    upstream purpletab_web {
        server web:80;
    }

    upstream purpletab_api {
        server controller:8080;
    }

    server {
        listen 80;
        server_name honker;

        client_max_body_size 100M;

        location /api/ {
            proxy_pass http://purpletab_api;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }

        location /content/ {
            proxy_pass http://purpletab_api;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
        }

        location / {
            proxy_pass http://purpletab_web;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
        }
    }
}
```

Add to `docker-compose.staging.yml`:

```yaml
services:
  nginx:
    image: nginx:alpine
    container_names: purpletab-nginx
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./ssl:/etc/nginx/ssl:ro  # Mount SSL certificates
    depends_on:
      - web
      - controller
    networks:
      - purpletab-network
    restart: unless-stopped
```

Note: For SSL with dockerized Nginx, you'll need to handle certificate provisioning separately (e.g., certbot, manual certs, or use external reverse proxy for SSL termination).

### Traefik (docker labels)

If your external Traefik is docker-based, add these labels to docker-compose.staging.yml:

```yaml
services:
  web:
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.purpletab-web.rule=Host(`honker`)"
      - "traefik.http.routers.purpletab-web.entrypoints=websecure"
      - "traefik.http.services.purpletab-web.loadbalancer.server.port=80"

  controller:
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.purpletab-api.rule=Host(`honker`) && (PathPrefix(`/api`) || PathPrefix(`/content`))"
      - "traefik.http.routers.purpletab-api.entrypoints=websecure"
      - "traefik.http.services.purpletab-api.loadbalancer.server.port=8080"
```

## Updating Deployment

```bash
# Pull latest changes
git pull

# Rebuild images
docker-compose -f docker-compose.yml -f docker-compose.staging.yml build

# Recreate containers
docker-compose -f docker-compose.yml -f docker-compose.staging.yml up -d

# View logs
docker-compose -f docker-compose.yml -f docker-compose.staging.yml logs -f
```

## Stopping Services

```bash
# Stop but keep data
docker-compose -f docker-compose.yml -f docker-compose.staging.yml down

# Stop and remove all data (⚠️  destructive)
docker-compose -f docker-compose.yml -f docker-compose.staging.yml down -v
```

## Troubleshooting

### Check service health
```bash
docker-compose -f docker-compose.yml -f docker-compose.staging.yml ps
```

### View logs for specific service
```bash
docker-compose -f docker-compose.yml -f docker-compose.staging.yml logs -f controller
```

### Test Ollama connectivity
```bash
docker exec purpletab-scraper-1 wget -O- http://100.64.0.2:11434/api/tags
```

### Access databases
```bash
# Controller DB
docker exec -it purpletab-controller-1 sqlite3 /app/data/controller.db

# Scraper DB
docker exec -it purpletab-scraper-1 sqlite3 /app/data/scraper.db
```
