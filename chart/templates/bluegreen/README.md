# Blue-Green Deployment Templates

This directory contains Argo Rollouts resources for blue-green deployments.

## What's Here

| File | Purpose |
|------|---------|
| `analysis-helm-tests.yaml` | Runs Helm tests as part of analysis |
| `analysis-health-checks.yaml` | Health checks and Prometheus metrics |
| `preview-services.yaml` | Preview services for blue/green environments |

## Activation

These templates are only rendered when `blueGreen.enabled: true` in values.yaml.

```yaml
blueGreen:
  enabled: true  # Activates these templates
```

## How It Works

1. **Preview Services**: Created for both blue and green environments
2. **Analysis Templates**: Define success/failure criteria
3. **Rollout Controller**: Argo Rollouts uses these templates during deployment

## Dependencies

Requires Argo Rollouts controller to be installed in the cluster.

See: [docs/ARGO-ROLLOUTS-SETUP.md](../../../../docs/ARGO-ROLLOUTS-SETUP.md)

## Testing

After enabling blue-green:

```bash
# Check resources are created
kubectl get analysistemplate -n docutag
kubectl get svc -n docutag | grep preview

# Perform test deployment
helm upgrade docutag ../../.. -n docutag

# Watch rollout
kubectl argo rollouts get rollout docutag-controller -n docutag --watch
```

## Analysis Flow

```
Deployment Starts
       ↓
Create Green Environment
       ↓
Wait for Pods Ready
       ↓
[Analysis Phase]
  ├─ Run Helm Tests (analysis-helm-tests.yaml)
  ├─ Check Health Endpoints (analysis-health-checks.yaml)
  └─ Check Prometheus Metrics (if enabled)
       ↓
  All Pass? ─[Yes]→ Promote to Production
       ↓ [No]
  Abort & Rollback
```

## Customization

Modify analysis behavior in values.yaml:

```yaml
blueGreen:
  analysis:
    helmVersion: "3.13.0"
    testTimeout: "180s"
    healthCheckDelay: "10s"
    prometheusEnabled: true
    errorRateThreshold: 0.05
```

## Related Documentation

- [Full Blue-Green Guide](../../../../docs/BLUE-GREEN-DEPLOYMENT.md)
- [Argo Rollouts Setup](../../../../docs/ARGO-ROLLOUTS-SETUP.md)
- [Helm Chart Tests](../tests/README.md)
