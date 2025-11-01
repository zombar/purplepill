# DocuTag Helm Chart

Official Helm chart for deploying DocuTag to Kubernetes.

## Overview

DocuTag is an AI-powered web content processing platform that scrapes web pages, extracts content, and performs comprehensive text analysis. This Helm chart packages all DocuTag services and dependencies for deployment to any Kubernetes cluster.

## Components

The chart deploys:

### Core Services
- **Controller** - Orchestration service with unified API (port 8080)
- **Scraper** - Web scraping with AI-powered extraction (port 8080)
- **TextAnalyzer** - Text analysis, sentiment, and NER (port 8080)
- **Scheduler** - Cron-based task scheduling (port 8080)
- **Web** - React-based web interface (port 80)

### Infrastructure Services
- **Redis** - Message queue backend (Bitnami chart)
- **PostgreSQL** - Database (Bitnami chart, optional)
- **Asynqmon** - Queue monitoring UI

### Observability Stack
- **Prometheus** - Metrics collection
- **Grafana** - Visualization dashboards
- **Loki** - Log aggregation
- **Tempo** - Distributed tracing
- **Promtail** - Log collector

### Ingress
- **Traefik IngressRoutes** - Traffic routing with automatic TLS

## Deployment Strategies

The chart supports multiple deployment strategies:

### Rolling Update (Default)
Standard Kubernetes rolling update with zero-downtime pod replacement.

### Progressive Delivery with Flagger (Recommended for Production)
Automated canary and blue-green deployments with metric-based validation using Flagger.

**Features**:
- ✅ Gradual traffic shifting (10% → 20% → ... → 100%)
- ✅ Automated Prometheus metrics monitoring
- ✅ Instant rollback on metric degradation
- ✅ Helm test validation via webhooks
- ✅ Zero-downtime deployments
- ✅ Optional Slack notifications

**Deployment Strategies**:

**Canary** - Gradual rollout with validation at each step:
```
Deploy → Run Tests → 10% traffic → 20% → ... → 100%
Total time: ~10-12 minutes
```

**Blue-Green** - Instant switch after full validation:
```
Deploy → Run Tests → Validate metrics → Switch 100% traffic
Total time: ~3-4 minutes
```

**Setup**: See [Flagger Setup Guide](../docs/FLAGGER-SETUP.md) and [Blue-Green Deployment Guide](../docs/BLUE-GREEN-DEPLOYMENT.md)

**Enable**:
```yaml
flagger:
  enabled: true
  strategy: canary  # or "blueGreen"
  analysis:
    interval: 1m
    stepWeight: 10  # 10% traffic per step
    iterations: 10  # 10 minutes total
  metrics:
    requestSuccessRate:
      enabled: true
      threshold: 99  # 99% success rate required
    requestDuration:
      enabled: true
      threshold: 500  # p99 latency < 500ms
```

**Prerequisites**: Flagger controller and loadtester installed in cluster. See [FLAGGER-SETUP.md](../docs/FLAGGER-SETUP.md).

### Automated Rollback Script (Simple Alternative)
For immediate rollback protection without Flagger infrastructure:

```bash
./scripts/deploy-with-rollback.sh docutag docutag values-production.yaml
```

Automatically rolls back if Helm tests fail after deployment. No additional infrastructure required.

## Prerequisites

- Kubernetes 1.24+
- Helm 3.8+
- Traefik ingress controller installed
- Storage class available (e.g., `do-block-storage` for DOKS)
- (Optional) Managed PostgreSQL database

## Installation

### Quick Start

```bash
# Add Bitnami repository for dependencies
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update

# Install with default values
helm install docutag ./chart
```

### Development Environment

```bash
helm install docutag ./chart -f ./chart/values-dev.yaml
```

### Staging Environment

```bash
helm install docutag ./chart \
  -f ./chart/values-staging.yaml \
  --set global.domain=docutag.honker
```

### Production Environment

```bash
helm install docutag ./chart \
  -f ./chart/values-production.yaml \
  --set global.domain=docutag.io \
  --set externalDatabase.host=your-db-host \
  --set-string externalDatabase.password=your-db-password
```

## Configuration

### Global Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `global.domain` | Domain name for the application | `docutag.local` |
| `global.imageRegistry` | Docker image registry | `ghcr.io/docutag` |
| `global.imagePullPolicy` | Image pull policy | `IfNotPresent` |
| `global.storageClass` | Storage class for PVCs | `do-block-storage` |

### Service Configuration

Each service supports:
- `enabled` - Enable/disable the service
- `replicaCount` - Number of replicas
- `image` - Image configuration
- `resources` - CPU/memory limits
- `autoscaling` - HPA configuration
- `env` - Environment variables

Example:

```yaml
controller:
  enabled: true
  replicaCount: 2
  autoscaling:
    enabled: true
    minReplicas: 2
    maxReplicas: 10
  resources:
    requests:
      memory: "256Mi"
      cpu: "250m"
```

### Database Options

#### Option 1: Bundled PostgreSQL (Development)

```yaml
postgresql:
  enabled: true
  auth:
    database: docutag
    username: docutag
    password: changeme
```

#### Option 2: External Managed Database (Production)

```yaml
postgresql:
  enabled: false

externalDatabase:
  enabled: true
  host: "db-postgresql-sfo3-12345.db.ondigitalocean.com"
  port: 25060
  database: docutag
  username: doadmin
  password: "your-password"
```

### Ingress Configuration

```yaml
ingress:
  enabled: true
  className: traefik
  tls:
    enabled: true
    certResolver: letsencrypt
  routes:
    web:
      host: "{{ .Values.global.domain }}"
      path: /
    api:
      host: "{{ .Values.global.domain }}"
      path: /api
    grafana:
      host: "{{ .Values.global.domain }}"
      path: /grafana
    asynqmon:
      # Asynqmon uses subdomain routing (doesn't support subpath)
      host: "asynqmon.{{ .Values.global.domain }}"
      path: /
```

**Note:** Asynqmon requires subdomain routing (e.g., `asynqmon.docutag.io`) because it doesn't properly support subpath deployment. Ensure DNS is configured with a wildcard A record or specific subdomain entry.

## Upgrading

```bash
# Upgrade to new version
helm upgrade docutag ./chart -f ./chart/values-staging.yaml

# Force pod restart
helm upgrade docutag ./chart --recreate-pods
```

## Testing

The chart includes built-in tests to verify the deployment:

```bash
# Run all tests
helm test docutag -n docutag

# Run with logs
helm test docutag -n docutag --logs
```

**Available tests:**
- Controller health endpoint
- Web UI accessibility
- Database connectivity (if PostgreSQL enabled)
- Redis connectivity (if Redis enabled)
- API endpoints (/health, /metrics, /api/sources)

See [chart/templates/tests/README.md](templates/tests/README.md) for detailed test documentation.

## Uninstallation

```bash
helm uninstall docutag
```

**Warning:** This will delete all resources. Ensure you have backups!

## Accessing Services

After installation:

1. **Web UI**: `https://your-domain/`
2. **API**: `https://your-domain/api/`
3. **Grafana**: `https://your-domain/grafana`
4. **Asynqmon**: `https://asynqmon.your-domain` (subdomain routing)

## Monitoring

### Grafana Dashboards

Access Grafana at `/grafana` to view:
- Service metrics
- Database performance
- Queue statistics
- System resources

### Prometheus Metrics

All services expose metrics at `/metrics` endpoint.

### Logs

View logs via Loki in Grafana or directly:

```bash
kubectl logs -n docutag deployment/docutag-controller
kubectl logs -n docutag deployment/docutag-scraper
```

## Troubleshooting

### Pods Not Starting

```bash
kubectl get pods -n docutag
kubectl describe pod <pod-name> -n docutag
kubectl logs <pod-name> -n docutag
```

### Database Connection Issues

```bash
kubectl exec -it deployment/docutag-controller -n docutag -- env | grep DB_
```

### Ingress Not Working

```bash
kubectl get ingressroute -n docutag
kubectl describe ingressroute docutag-web -n docutag
```

## Values Files

- `values.yaml` - Default configuration
- `values-dev.yaml` - Development overrides
- `values-staging.yaml` - Staging overrides
- `values-production.yaml` - Production overrides

## Architecture

```
Internet → Traefik Ingress → Services
                              ├── Web (Frontend)
                              ├── Controller (API)
                              ├── Scraper
                              ├── TextAnalyzer
                              └── Scheduler
                                   ↓
                            PostgreSQL + Redis
```

## Resource Requirements

### Minimum (Development)
- 2 vCPUs
- 4 GB RAM
- 20 GB storage

### Recommended (Staging)
- 4 vCPUs
- 8 GB RAM
- 100 GB storage

### Production
- 8+ vCPUs
- 16+ GB RAM
- 200+ GB storage

## Security

- All services run as non-root users
- Read-only root filesystems where possible
- Network policies (optional)
- Secret management via Kubernetes secrets

## License

MIT License - see LICENSE file

## Support

- GitHub Issues: https://github.com/docutag/platform/issues
- Documentation: https://github.com/docutag/platform
