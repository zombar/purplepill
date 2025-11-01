# Flagger Progressive Delivery Templates

This directory contains Flagger resources for automated canary and blue-green deployments.

## Overview

Flagger automates the promotion of canary deployments using:
- **Traefik** for traffic routing (already integrated)
- **Prometheus** for metrics analysis  (already deployed)
- **Helm tests** via webhook for validation

## Files

| File | Purpose |
|------|---------|
| `canary-controller.yaml` | Progressive delivery for controller service |
| `canary-web.yaml` | Progressive delivery for web UI |
| `canary-scraper.yaml` | Progressive delivery for scraper service |
| `canary-textanalyzer.yaml` | Progressive delivery for textanalyzer service |
| `canary-scheduler.yaml` | Progressive delivery for scheduler service |
| `metric-templates.yaml` | Prometheus query templates for metrics |
| `alert-provider.yaml` | Slack notifications (optional) |

## Activation

These templates are only rendered when `flagger.enabled: true`:

```yaml
flagger:
  enabled: true  # Activates all Flagger resources
  strategy: canary  # or "blueGreen"
```

## How It Works

### Canary Strategy (Gradual Rollout)

```
1. helm upgrade (new image version)
   ↓
2. Flagger detects new ReplicaSet
   ↓
3. Run Helm tests (webhook)
   ↓
4. Tests pass? → Start traffic shift
   ↓
5. Traffic progression:
   Old: 100% → 90% → 80% → ... → 0%
   New:   0% → 10% → 20% → ... → 100%
   ↓
6. At each step, monitor Prometheus metrics:
   - Request success rate > 99%
   - Request duration p99 < 500ms
   ↓
7. Metrics OK? → Continue to next step
   Metrics fail? → Instant rollback
   ↓
8. After 100% traffic → Promote canary
   ↓
9. Scale down old version
```

### Blue-Green Strategy (Instant Switch)

```
1. helm upgrade (new image version)
   ↓
2. Flagger creates new (green) ReplicaSet
   ↓
3. Run Helm tests
   ↓
4. Monitor metrics with 0% production traffic
   ↓
5. All checks pass? → Switch 100% traffic
   All checks fail? → Delete green, keep blue
   ↓
6. Scale down old (blue) version
```

## Canary Resources

Each service gets a Canary resource that defines:

**targetRef**: Points to existing Deployment (e.g., `docutag-controller`)

**service**: Port configuration for routing

**analysis**:
- `interval`: How often to check metrics (default: 1m)
- `threshold`: Failed checks before rollback (default: 5)
- `stepWeight`: Traffic increase per step (default: 10%)
- `maxWeight`: Maximum canary weight (default: 100%)

**metrics**:
- Request success rate (from Prometheus)
- Request duration p99 latency (from Prometheus)
- Custom error rate (optional)

**webhooks**:
- Helm tests (pre-rollout validation)
- Slack notifications (deployment events)

## MetricTemplates

Prometheus queries used by Flagger:

### request-success-rate
Percentage of successful requests (non-5xx):
```promql
sum(rate(http_requests_total{status!~"5.."}))
/
sum(rate(http_requests_total)) * 100
```

### request-duration
99th percentile latency in milliseconds:
```promql
histogram_quantile(0.99,
  sum(rate(http_request_duration_milliseconds_bucket)) by (le)
)
```

### error-rate (optional)
Percentage of error responses (4xx + 5xx):
```promql
sum(rate(http_requests_total{status=~"[45].."}))
/
sum(rate(http_requests_total)) * 100
```

## Traefik Integration

Flagger automatically manages Traefik routing:

**Creates services**:
- `docutag-controller` - Stable endpoint
- `docutag-controller-primary` - Current production version
- `docutag-controller-canary` - New version being tested

**Traffic management**:
- Flagger adjusts weights between primary/canary
- Existing IngressRoutes route to stable service
- Transparent to external clients

## Monitoring Deployments

### CLI Commands

```bash
# List all canaries
kubectl get canaries -n docutag

# Watch specific canary
kubectl get canary docutag-controller -n docutag -w

# Describe canary (see events and status)
kubectl describe canary docutag-controller -n docutag

# Manual promotion
kubectl flagger promote docutag-controller -n docutag

# Manual rollback
kubectl flagger rollback docutag-controller -n docutag
```

### Canary Status

```bash
$ kubectl get canary docutag-controller -n docutag

NAME                 STATUS      WEIGHT   LASTTRANSITIONTIME
docutag-controller   Progressing 30       2025-11-01T22:15:00Z
```

**Status values**:
- `Initializing` - Creating resources
- `Initialized` - Ready for analysis
- `Progressing` - Traffic shifting in progress
- `Promoting` - Finalizing promotion
- `Succeeded` - Deployment successful
- `Failed` - Rolled back due to failures

### Events

```bash
$ kubectl describe canary docutag-controller -n docutag

Events:
  Type    Reason  Age   Message
  ----    ------  ----  -------
  Normal  Synced  3m    New revision detected! Scaling up docutag-controller-canary
  Normal  Synced  2m    Starting canary analysis for docutag-controller
  Normal  Synced  2m    Advance docutag-controller canary weight 10
  Normal  Synced  1m    Advance docutag-controller canary weight 20
  Normal  Synced  30s   Advance docutag-controller canary weight 30
```

## Configuration Examples

### Canary with Custom Thresholds

```yaml
flagger:
  enabled: true
  strategy: canary
  analysis:
    interval: 30s  # Check every 30 seconds
    threshold: 3   # Rollback after 3 failures
    stepWeight: 20 # 20% traffic per step
  metrics:
    requestSuccessRate:
      enabled: true
      threshold: 95  # 95% success rate minimum
```

### Blue-Green for Critical Services

```yaml
flagger:
  enabled: true
  strategy: blueGreen  # Instant switch
  analysis:
    interval: 1m
    threshold: 5
  webhooks:
    helmTests:
      enabled: true
      timeout: 5m  # Longer timeout for critical tests
```

## Troubleshooting

### Canary Stuck in Progressing

```bash
# Check metric values
kubectl logs -n flagger-system deployment/flagger -f | grep docutag-controller

# Common causes:
# - Metrics below threshold
# - Prometheus unreachable
# - Metric query returning no data
```

### Automatic Rollback

```bash
# View rollback reason in events
kubectl describe canary docutag-controller -n docutag

# Common reasons:
# - Request success rate < 99%
# - Request duration > 500ms
# - Helm tests failed
```

### Manual Intervention

```bash
# Skip analysis and promote immediately
kubectl flagger promote docutag-controller -n docutag

# Abort and rollback
kubectl flagger rollback docutag-controller -n docutag
```

## Dependencies

**Required**:
- Flagger controller installed in cluster
- Prometheus with metrics from services
- Services exposing `/metrics` endpoints

**Optional**:
- flagger-loadtester (for Helm test webhooks)
- Slack webhook (for notifications)

## Related Documentation

- [Flagger Setup Guide](../../../../docs/FLAGGER-SETUP.md)
- [Blue-Green Deployment](../../../../docs/BLUE-GREEN-DEPLOYMENT.md)
- [Helm Chart Tests](../tests/README.md)
- [Flagger Documentation](https://docs.flagger.app/)
