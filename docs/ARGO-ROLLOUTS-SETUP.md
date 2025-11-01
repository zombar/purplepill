# Argo Rollouts Setup for Blue-Green Deployments

Quick setup guide for enabling Argo Rollouts blue-green deployments in DocuTag.

## Prerequisites

- Kubernetes cluster with admin access
- kubectl configured
- Helm 3.x installed

## Installation Steps

### 1. Install Argo Rollouts Controller

```bash
# Create namespace
kubectl create namespace argo-rollouts

# Install Argo Rollouts
kubectl apply -n argo-rollouts -f https://github.com/argoproj/argo-rollouts/releases/latest/download/install.yaml

# Verify installation
kubectl get pods -n argo-rollouts

# Expected output:
# NAME                             READY   STATUS    RESTARTS   AGE
# argo-rollouts-xxx-yyy            1/1     Running   0          30s
```

### 2. Install kubectl Plugin (Optional)

The kubectl plugin provides CLI control over rollouts:

```bash
# Linux
curl -LO https://github.com/argoproj/argo-rollouts/releases/latest/download/kubectl-argo-rollouts-linux-amd64
chmod +x kubectl-argo-rollouts-linux-amd64
sudo mv kubectl-argo-rollouts-linux-amd64 /usr/local/bin/kubectl-argo-rollouts

# macOS
curl -LO https://github.com/argoproj/argo-rollouts/releases/latest/download/kubectl-argo-rollouts-darwin-amd64
chmod +x kubectl-argo-rollouts-darwin-amd64
sudo mv kubectl-argo-rollouts-darwin-amd64 /usr/local/bin/kubectl-argo-rollouts

# Verify
kubectl argo rollouts version
```

### 3. Enable Blue-Green in Helm Chart

**For production** (`chart/values-production.yaml`):

```yaml
blueGreen:
  enabled: true
  autoPromote: false  # Require manual promotion
  scaleDownDelay: 300  # Keep old version for 5 minutes
  analysis:
    prometheusEnabled: true  # Enable Prometheus metrics
```

**For staging** (`chart/values-staging.yaml`):

```yaml
blueGreen:
  enabled: true
  autoPromote: true  # Auto-promote in staging
  scaleDownDelay: 60
```

### 4. Update Deployments to Rollouts

The chart already includes Rollout templates that activate when `blueGreen.enabled: true`. No additional changes needed!

### 5. Deploy with Blue-Green

```bash
# Deploy or upgrade
helm upgrade --install docutag ./chart -n docutag -f ./chart/values-production.yaml

# Watch rollout progress
kubectl argo rollouts get rollout docutag-controller -n docutag --watch

# Output:
# Name:            docutag-controller
# Namespace:       docutag
# Status:          ॥ Paused
# Strategy:        BlueGreen
# Images:          ghcr.io/docutag/docutag-controller:1.1.0 (green)
#                  ghcr.io/docutag/docutag-controller:1.0.0 (blue, active)
# Replicas:
#   Desired:       3
#   Current:       6 (3 blue + 3 green)
#   Updated:       3
#   Ready:         6
#   Available:     6
#
# Analysis Runs:
#   ✓ helm-tests (Successful)
#   ⋮ health-checks (Running)
```

### 6. Promote or Abort

```bash
# If autoPromote: false, manually promote after analysis passes
kubectl argo rollouts promote docutag-controller -n docutag

# Or abort if issues detected
kubectl argo rollouts abort docutag-controller -n docutag
```

## Dashboard (Optional)

Argo Rollouts provides a web UI:

```bash
# Start dashboard
kubectl argo rollouts dashboard -n docutag

# Access at http://localhost:3100
```

The dashboard shows:
- Current rollout status
- Blue and green ReplicaSets
- Analysis progress
- Promotion button

## Verification

After enabling blue-green, verify it's working:

```bash
# 1. Check Rollout resource exists
kubectl get rollouts -n docutag

# 2. Perform a test deployment
helm upgrade docutag ./chart -n docutag -f values-production.yaml

# 3. Watch rollout
kubectl argo rollouts get rollout docutag-controller -n docutag --watch

# 4. Verify analysis runs
kubectl get analysisrun -n docutag

# 5. Check logs
kubectl logs -n argo-rollouts deployment/argo-rollouts
```

## Troubleshooting

### Rollout Stuck in Paused State

```bash
# Check analysis status
kubectl get analysisrun -n docutag

# View analysis logs
kubectl logs -n docutag analysisrun/<name>

# Common causes:
# - Helm tests failing
# - Health checks timing out
# - Prometheus metrics unavailable
```

### Analysis Fails Immediately

```bash
# Check if preview services exist
kubectl get svc -n docutag | grep preview

# Verify pods are ready
kubectl get pods -n docutag

# Check analysis template
kubectl describe analysistemplate docutag-helm-tests -n docutag
```

### Rollback Not Working

```bash
# Manually abort
kubectl argo rollouts abort docutag-controller -n docutag

# Force rollback
kubectl argo rollouts undo docutag-controller -n docutag
```

## Uninstalling Argo Rollouts

```bash
# Delete from cluster
kubectl delete -n argo-rollouts -f https://github.com/argoproj/argo-rollouts/releases/latest/download/install.yaml

# Delete namespace
kubectl delete namespace argo-rollouts
```

## Migration Back to Standard Deployments

To disable blue-green and revert to standard deployments:

```yaml
# values-production.yaml
blueGreen:
  enabled: false
```

Then upgrade:
```bash
helm upgrade docutag ./chart -n docutag -f values-production.yaml
```

The chart will automatically revert to using standard Kubernetes Deployments.

## Best Practices

1. **Enable in production first** - Staging can use faster rolling updates
2. **Start with autoPromote: false** - Require manual approval until comfortable
3. **Monitor first few deployments** - Watch dashboard and logs closely
4. **Set appropriate delays** - Give enough time for traffic validation
5. **Enable Prometheus analysis** - Better validation of deployment health
6. **Test abort procedure** - Practice aborting and rolling back

## Related Documentation

- [Blue-Green Deployment Strategy](./BLUE-GREEN-DEPLOYMENT.md) - Comprehensive guide
- [Argo Rollouts Docs](https://argo-rollouts.readthedocs.io/)
- [Helm Chart Tests](../chart/templates/tests/README.md)
