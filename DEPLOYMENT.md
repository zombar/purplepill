# Staging Deployment Guide

## Prerequisites

1. **External Reverse Proxy** configured to route:
   - `http://honker/purpletab/*` → `localhost:3001` (Web UI)
   - `http://honker/purpletab/api/*` → `localhost:9080` (Controller API)
   - `http://honker/purpletab/content/*` → `localhost:9080` (SEO content)

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
    # Forward API requests, stripping /purpletab prefix before sending to controller
    handle_path /purpletab/api/* {
        rewrite * /api{uri}
        reverse_proxy localhost:9080
    }
    # Forward content requests, stripping /purpletab prefix
    handle_path /purpletab/content/* {
        rewrite * /content{uri}
        reverse_proxy localhost:9080
    }
    # Forward web UI requests, keeping /purpletab prefix
    handle /purpletab/* {
        reverse_proxy localhost:3001
    }
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

# HTTP server
server {
    listen 80;
    server_name honker;

    # Security Headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "no-referrer-when-downgrade" always;

    # IMPORTANT: Do NOT set a restrictive Content-Security-Policy header here
    # The React app needs to load scripts and stylesheets normally
    # If you need CSP, use something like:
    # add_header Content-Security-Policy "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline';" always;

    # Logging
    access_log /var/log/nginx/purpletab-access.log;
    error_log /var/log/nginx/purpletab-error.log;

    # Client limits
    client_max_body_size 100M;
    client_body_timeout 120s;

    # API routes - strip /purpletab prefix
    location /purpletab/api/ {
        rewrite ^/purpletab(/api/.*)$ $1 break;
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

    # SEO content routes - strip /purpletab prefix
    location /purpletab/content/ {
        rewrite ^/purpletab(/content/.*)$ $1 break;
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
    location /purpletab/health {
        rewrite ^/purpletab(/health)$ $1 break;
        proxy_pass http://purpletab_api;
        access_log off;
        proxy_set_header Host $host;
    }

    # Web UI - catch all purpletab requests
    # Forward with /purpletab prefix intact
    location /purpletab/ {
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
    location /purpletab/api/ {
        limit_req zone=api_limit burst=20 nodelay;
        limit_conn addr 10;

        rewrite ^/purpletab(/api/.*)$ $1 break;
        proxy_pass http://purpletab_api;
        # ... rest of proxy config ...
    }

    # SEO content with rate limiting
    location /purpletab/content/ {
        limit_req zone=content_limit burst=50 nodelay;

        rewrite ^/purpletab(/content/.*)$ $1 break;
        proxy_pass http://purpletab_api;
        # ... rest of proxy config ...
    }

    # Web UI with rate limiting
    location /purpletab/ {
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
    location /purpletab/content/ {
        proxy_cache purpletab_cache;
        proxy_cache_valid 200 60m;
        proxy_cache_valid 404 10m;
        proxy_cache_use_stale error timeout updating http_500 http_502 http_503 http_504;
        proxy_cache_background_update on;
        proxy_cache_lock on;

        add_header X-Cache-Status $upstream_cache_status;

        rewrite ^/purpletab(/content/.*)$ $1 break;
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

        location /purpletab/api/ {
            rewrite ^/purpletab(/api/.*)$ $1 break;
            proxy_pass http://purpletab_api;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }

        location /purpletab/content/ {
            rewrite ^/purpletab(/content/.*)$ $1 break;
            proxy_pass http://purpletab_api;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
        }

        location /purpletab/ {
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
      - "traefik.http.routers.purpletab-web.rule=Host(`honker`) && PathPrefix(`/purpletab`)"
      - "traefik.http.routers.purpletab-web.entrypoints=web"
      - "traefik.http.services.purpletab-web.loadbalancer.server.port=80"

  controller:
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.purpletab-api.rule=Host(`honker`) && (PathPrefix(`/purpletab/api`) || PathPrefix(`/purpletab/content`))"
      - "traefik.http.routers.purpletab-api.entrypoints=web"
      - "traefik.http.services.purpletab-api.loadbalancer.server.port=8080"
      # Strip /purpletab prefix before forwarding to controller
      - "traefik.http.middlewares.purpletab-strip.stripprefix.prefixes=/purpletab"
      - "traefik.http.routers.purpletab-api.middlewares=purpletab-strip"
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
