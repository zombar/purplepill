# Deployment Guide

## Quick Start - GitHub Registry Workflow

### 1. Build and Push (Dev Machine)

```bash
# Build all service images and push to GitHub Container Registry
make docker-staging-push
```

### 2. Deploy (Staging Server)

```bash
# Pull latest images and start services
make docker-staging-pull
```

That's it! This pulls all 5 service images from ghcr.io/zombar and starts them.

## Staging URL

The application is hosted at: **https://purpletab.honker**

Your external reverse proxy (Caddy, Nginx, Traefik, etc.) should route:
- `purpletab.honker` → `localhost:3001` (web container)

The web container's nginx handles:
- Static files (HTML, JS, CSS, images)
- API proxying: `/api/*` → `controller:8080` (internal)

## First Time Setup

### GitHub Authentication

Before pushing or pulling images, authenticate with GitHub Container Registry:

```bash
# Create a GitHub Personal Access Token with 'write:packages' scope
# at https://github.com/settings/tokens

# Login to GitHub Container Registry
echo YOUR_GITHUB_TOKEN | docker login ghcr.io -u zombar --password-stdin
```

You only need to do this once per machine.

## Available Commands

### Dev Machine

```bash
# Using Makefile (recommended)
make docker-staging-build   # Build all service images
make docker-staging-push    # Build and push all to ghcr.io/zombar
make docker-staging-deploy  # Full local deploy (build + start)

# Using scripts directly
./build-staging.sh          # Build only
./build-staging.sh push     # Build and push to registry
```

### Staging Server

```bash
# Using Makefile (recommended)
make docker-staging-pull    # Pull latest images and start services
make docker-staging-up      # Start services (without pulling)
make docker-staging-down    # Stop services
make docker-staging-logs    # View logs

# Using scripts directly
./deploy-staging.sh         # Pull latest images and start services
```

## Built Images

All services are built as multi-platform staging images (linux/amd64, linux/arm64) and pushed to GitHub Container Registry:

- `ghcr.io/zombar/purpletab-textanalyzer:staging`
- `ghcr.io/zombar/purpletab-scraper:staging`
- `ghcr.io/zombar/purpletab-controller:staging`
- `ghcr.io/zombar/purpletab-scheduler:staging`
- `ghcr.io/zombar/purpletab-web:staging`

Multi-platform support ensures images work on both ARM64 (Apple Silicon) and AMD64 (Intel/AMD) servers.

## Configuration Files

- **docker-compose.yml** - Base configuration (all services)
- **docker-compose.staging.yml** - Staging overrides (uses GitHub registry images)
- **docker-compose.build-staging.yml** - Build configuration for staging images
- **build-staging.sh** - Build and push script (dev machine)
- **deploy-staging.sh** - Pull and deploy script (staging server)

## Staging Configuration

The application is configured for subdomain hosting at `purpletab.honker`:

- **Web service**: Built with `VITE_PUBLIC_URL_BASE=/` (root path)
- **API calls**: Same-origin requests to `/api/*`
- **CORS**: Disabled (not needed for same-origin)
- **Nginx**: Proxies `/api/*` to controller service internally

This is much simpler than subdirectory hosting - no path rewriting or base path handling needed.

## External Reverse Proxy Setup

Configure your reverse proxy to route the subdomain to the web container. Examples:

### Caddy

```
purpletab.honker {
    reverse_proxy localhost:3001
}
```

### Nginx

```nginx
server {
    listen 443 ssl http2;
    server_name purpletab.honker;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://localhost:3001;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
    }
}
```

### Traefik

```yaml
http:
  routers:
    purpletab:
      rule: "Host(`purpletab.honker`)"
      service: purpletab
      tls:
        certResolver: myresolver

  services:
    purpletab:
      loadBalancer:
        servers:
          - url: "http://localhost:3001"
```

## Troubleshooting

### Authentication Issues

If you get authentication errors:

```bash
# Re-login to GitHub Container Registry
docker logout ghcr.io
echo YOUR_GITHUB_TOKEN | docker login ghcr.io -u zombar --password-stdin
```

### Image Pull Failures

If images fail to pull on the server:

```bash
# Check if you're logged in
docker login ghcr.io

# Manually pull a specific image to test
docker pull ghcr.io/zombar/purpletab-web:staging
```

### View Running Services

```bash
docker compose -f docker-compose.yml -f docker-compose.staging.yml ps
```

### Platform Mismatch Warnings

If you see warnings like "requested image's platform (linux/arm64) does not match the detected host platform (linux/amd64)", your images were built without multi-platform support.

To fix:
1. Rebuild and push with multi-platform support:
   ```bash
   make docker-staging-push
   ```
2. Pull on the server:
   ```bash
   make docker-staging-pull
   ```

The updated build script automatically creates multi-platform images that work on both ARM64 and AMD64.

## Future: CI/CD Integration

This manual workflow is designed to be easily automated with GitHub Actions in the future. The build-staging.sh and deploy-staging.sh scripts can be called directly from CI pipelines.
