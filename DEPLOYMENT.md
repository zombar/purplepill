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

All services are built as staging images and pushed to GitHub Container Registry:

- `ghcr.io/zombar/purpletab-textanalyzer:staging`
- `ghcr.io/zombar/purpletab-scraper:staging`
- `ghcr.io/zombar/purpletab-controller:staging`
- `ghcr.io/zombar/purpletab-scheduler:staging`
- `ghcr.io/zombar/purpletab-web:staging`

## Configuration Files

- **docker-compose.yml** - Base configuration (all services)
- **docker-compose.staging.yml** - Staging overrides (uses GitHub registry images)
- **docker-compose.build-staging.yml** - Build configuration for staging images
- **build-staging.sh** - Build and push script (dev machine)
- **deploy-staging.sh** - Pull and deploy script (staging server)

## Staging Configuration

The web service is built with:
- VITE_PUBLIC_URL_BASE=http://honker/purpletab
- VITE_CONTROLLER_API_URL=/purpletab

The nginx proxy forwards /api/* requests to the controller service.

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
docker-compose -f docker-compose.yml -f docker-compose.staging.yml ps
```

## Future: CI/CD Integration

This manual workflow is designed to be easily automated with GitHub Actions in the future. The build-staging.sh and deploy-staging.sh scripts can be called directly from CI pipelines.
