# Progressive Delivery for DocuTag

This document describes progressive delivery strategies for DocuTag production deployments with automated rollback.

## Overview

DocuTag supports two progressive delivery approaches:

1. **Flagger** (Recommended) - Automated canary/blue-green with Prometheus monitoring
2. **Deploy Script** (Simple) - Automated rollback script for immediate protection

## Option 1: Flagger (Recommended for Production)

### What is Flagger?

Flagger automates progressive delivery using your existing infrastructure:
- ✅ **Traefik** for traffic routing (already integrated)
- ✅ **Prometheus** for metrics monitoring (already deployed)
- ✅ **Helm tests** for validation (already exist)
- ✅ **Automated rollback** on metric degradation

### Why Flagger?

| Feature | Benefit |
|---------|---------|
| **Gradual Rollout** | Expose new version to 10% traffic first |
| **Automatic Rollback** | Instant rollback if metrics degrade |
| **Zero Downtime** | Old and new versions run simultaneously |
| **Proven** | Used by Weaveworks, GitLab, Microsoft |
| **Flexible** | Supports both canary and blue-green |

### Deployment Strategies

#### Canary (Gradual Rollout)

Progressive traffic shift with validation at each step:

```
Deploy v1.1.0
  ↓
Run Helm Tests
  ↓
Traffic: 10% new, 90% old
Monitor metrics (1 min)
  ↓
Metrics OK? → 20% new, 80% old
Metrics fail? → Instant rollback
  ↓
... continue to 100%
  ↓
Promote and cleanup

Total time: ~10-12 minutes
```

**Benefits**:
- Early detection with limited blast radius
- Gradual confidence building
- Automatic rollback at any step

**Use when**:
- Deploying significant changes
- High traffic services
- Want maximum safety

#### Blue-Green (Instant Switch)

Full validation then instant traffic switch:

```
Deploy v1.1.0 (green)
  ↓
Run Helm Tests
Monitor metrics with 0% traffic
  ↓
All checks pass? → Switch 100% traffic
All checks fail? → Delete green, keep blue
  ↓
Cleanup old version

Total time: ~3-4 minutes
```

**Benefits**:
- Faster deployment
- Simple instant rollback
- Good for validated releases

**Use when**:
- Deploying minor updates
- Changes already tested extensively
- Want faster deployment

### Setup

See [FLAGGER-SETUP.md](./FLAGGER-SETUP.md) for complete installation guide.

**Quick setup**:

```bash
# 1. Install Flagger (one-time)
helm repo add flagger https://flagger.app
helm upgrade -i flagger flagger/flagger \
  --namespace flagger-system \
  --create-namespace \
  --set prometheus.install=false \
  --set meshProvider=traefik

# 2. Install loadtester (for Helm test webhooks)
helm upgrade -i flagger-loadtester flagger/loadtester \
  --namespace flagger-system

# 3. Enable in production
# Set flagger.enabled: true in values-production.yaml

# 4. Deploy
cd infra
pulumi config set imageVersion 1.1.0 --stack production
pulumi up

# 5. Monitor rollout
kubectl get canaries -n docutag -w
```

### Configuration

**File**: `chart/values-production.yaml`

```yaml
flagger:
  enabled: true
  strategy: canary  # or "blueGreen"

  analysis:
    interval: 1m       # Check metrics every minute
    threshold: 5       # Rollback after 5 failures
    stepWeight: 10     # 10% traffic per step
    iterations: 10     # 10 minutes total

  metrics:
    requestSuccessRate:
      enabled: true
      threshold: 99    # 99% success rate required

    requestDuration:
      enabled: true
      threshold: 500   # p99 < 500ms

  webhooks:
    helmTests:
      enabled: true
      timeout: 3m
```

### How It Works

**Architecture**:

```
┌─────────────────────┐
│   Traefik Ingress   │
│  (eng.in.docutag.app)│
└──────────┬──────────┘
           │
           ├─ 90% traffic ──┐
           │                 ▼
           │         ┌──────────────────┐
           │         │  Primary (old)   │
           │         │  controller:1.0.0 │
           │         └──────────────────┘
           │
           ├─ 10% traffic ──┐
           │                 ▼
           │         ┌──────────────────┐
           │         │  Canary (new)    │
           │         │  controller:1.1.0 │
           │         └──────────────────┘
           │
           ▼
    ┌─────────────────────────┐
    │   Prometheus Metrics    │
    │  - Success rate: 99.5%  │
    │  - Latency p99: 450ms   │
    └─────────────────────────┘
            │
            ▼
    ┌─────────────────────────┐
    │   Flagger Analysis      │
    │  Metrics OK? Continue   │
    │  Metrics bad? Rollback  │
    └─────────────────────────┘
```

**Monitoring**:

```bash
# Watch canary status
kubectl get canary docutag-controller -n docutag -w

# Output:
# NAME                 STATUS      WEIGHT   LASTTRANSITIONTIME
# docutag-controller   Progressing 30       2025-11-01T22:30:00Z
```

**Manual control**:

```bash
# Promote immediately (skip remaining steps)
kubectl flagger promote docutag-controller -n docutag

# Abort and rollback
kubectl flagger rollback docutag-controller -n docutag
```

### Advantages

✅ **Automated**: No manual intervention needed
✅ **Safe**: Gradual rollout limits blast radius
✅ **Fast rollback**: Instant traffic shift on failure
✅ **Observability**: Built-in metrics monitoring
✅ **Battle-tested**: Production-ready tool

### Cost

**Additional resources during deployment**:
- 2x pods running (old + new)
- Duration: 10-12 minutes (canary) or 3-4 minutes (blue-green)
- Cost impact: Minimal (only during upgrades)

---

## Option 2: Deploy-with-Rollback Script (Simple Alternative)

If you want automated rollback **without** installing Flagger:

### What It Does

Automated rollback script that:
1. Captures current revision before upgrade
2. Performs `helm upgrade`
3. Waits for rollout completion
4. Runs Helm tests
5. **If tests fail** → automatic rollback

### Setup

No installation required! Script is already in repo.

### Usage

```bash
./scripts/deploy-with-rollback.sh docutag docutag chart/values-production.yaml
```

**Output**:
```
==========================================
Helm Deploy with Automated Rollback
==========================================

ℹ Validating prerequisites...
✓ Prerequisites validated
ℹ Deploying docutag to namespace docutag...
ℹ Upgrading from revision 5 (version 1.0.0)
ℹ New revision: 6 (version 1.1.0)
⏳ Waiting for rollout to complete...
✓ All deployments rolled out successfully
🧪 Running Helm tests...
✓ All tests passed!
✅ Deployment completed successfully!
```

### Rollback on Failure

If tests fail:

```
🧪 Running Helm tests...
✗ Tests failed!
✗ Deployment validation failed!
⚠ Rolling back to revision 5 (version 1.0.0)...
✓ Rollback to revision 5 completed successfully
```

### Advantages

✅ **No infrastructure changes** needed
✅ **Simple** bash script
✅ **Immediate** availability
✅ **Zero cost** additional resources

### Limitations

❌ **No gradual rollout** (all-or-nothing)
❌ **Rollback takes time** (~30-60 seconds)
❌ **Brief downtime** during rollback
❌ **No metric monitoring** (only Helm tests)

---

## Comparison

| Feature | Flagger | Deploy Script |
|---------|---------|---------------|
| **Rollback Time** | Instant (<1s) | 30-60 seconds |
| **Downtime** | Zero | Brief during rollback |
| **Gradual Rollout** | ✅ Yes | ❌ No |
| **Metric Monitoring** | ✅ Prometheus | ❌ Tests only |
| **Setup Complexity** | Medium | None |
| **Cost Impact** | 2x pods (10min) | None |
| **Production Ready** | ✅ Yes | ✅ Yes |

## Recommendation

**For Production**: Use **Flagger**

- True zero-downtime deployments
- Gradual rollout catches issues early
- Prometheus metrics validation
- Instant rollback
- Industry standard

**For Quick Start**: Use **Deploy Script**

- Get automated rollback today
- No new infrastructure
- Migrate to Flagger later

## Migration Path

**Phase 1** (Week 1): Deploy Script
```bash
# Use script for automated rollback
./scripts/deploy-with-rollback.sh docutag docutag values-production.yaml
```

**Phase 2** (Week 2-3): Install Flagger
```bash
# Install Flagger controller
helm install flagger flagger/flagger -n flagger-system ...

# Test in non-critical service first
# Verify metrics collection
# Practice rollback scenarios
```

**Phase 3** (Week 4+): Enable in Production
```yaml
# values-production.yaml
flagger:
  enabled: true
  strategy: canary
```

## Troubleshooting

### Flagger Issues

See [FLAGGER-SETUP.md#troubleshooting](./FLAGGER-SETUP.md#troubleshooting)

Common issues:
- Canary stuck: Check targetRef and service config
- Auto rollback: Check Prometheus metrics
- Metrics unavailable: Verify Prometheus address

### Deploy Script Issues

**Script fails with "helm not found"**:
```bash
# Install Helm
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
```

**Script fails with "kubectl not found"**:
```bash
# Install kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
chmod +x kubectl
sudo mv kubectl /usr/local/bin/
```

**Tests timeout**:
```bash
# Increase timeout in script (line 20)
TIMEOUT_TESTS="300s"  # 5 minutes
```

## Best Practices

### 1. Start Conservative

Begin with strict thresholds:
```yaml
metrics:
  requestSuccessRate:
    threshold: 99  # High bar
analysis:
  threshold: 3     # Low tolerance for failures
```

### 2. Monitor First Deployments

Watch the first few closely:
```bash
# Terminal 1: Canary status
kubectl get canaries -n docutag -w

# Terminal 2: Flagger logs
kubectl -n flagger-system logs -f deployment/flagger
```

### 3. Test Rollback

Practice before you need it:
```bash
# Deploy with intentional failure
# Verify automatic rollback works
# Document rollback time
```

### 4. Enable Notifications

Stay informed:
```yaml
alerts:
  slack:
    enabled: true
    webhookUrl: "..."
    channel: "#production-deploys"
```

### 5. Tune for Your SLOs

Adjust thresholds based on actual performance:
```yaml
# If your service normally has 99.9% success
metrics:
  requestSuccessRate:
    threshold: 99.5  # Set slightly lower
```

## Related Documentation

- [Flagger Setup Guide](./FLAGGER-SETUP.md) - Complete installation instructions
- [Flagger Templates](../chart/templates/flagger/README.md) - Template documentation
- [Helm Tests](../chart/templates/tests/README.md) - Test suite details
- [Production Config Verification](./PRODUCTION-CONFIG-VERIFICATION.md) - Pre-deployment checks

## References

- [Flagger Documentation](https://docs.flagger.app/)
- [Flagger + Traefik Guide](https://docs.flagger.app/tutorials/traefik-progressive-delivery)
- [Progressive Delivery](https://redmonk.com/jgovernor/2018/08/06/towards-progressive-delivery/)
