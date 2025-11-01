# CI/CD Setup Guide

Complete guide for setting up automated infrastructure and application deployment workflows.

## Overview

The CI/CD pipeline is split into three main workflows:

1. **Infrastructure** (`infrastructure.yml`) - Manages cluster, Traefik, Flagger, DNS via Pulumi
2. **Release** (`release.yml`) - Builds images, creates releases on merge to main
3. **Application Deployment** (`production-deploy.yml`) - Deploys application versions to Kubernetes

## Architecture

```
Merge to main
  ↓
Release Workflow (automatic)
  ├── Calculate version (semver from commits)
  ├── Build & push Docker images
  └── Create GitHub release
  ↓
Application Deploy Workflow (automatic)
  ├── Extract version from release
  ├── Deploy via Helm
  ├── Run tests
  └── Monitor Flagger rollout

Infrastructure changes
  ↓
Infrastructure Workflow (manual/automatic)
  ├── Deploy via Pulumi
  ├── Create/update cluster
  ├── Deploy Traefik, Flagger
  └── Configure DNS
```

## Required Secrets

**Recommended approach**: Use **organization-level secrets** for simplicity.

See **[GitHub Secrets Setup Guide](./GITHUB-SECRETS-SETUP.md)** for detailed instructions.

### Quick Setup (4 Organization Secrets)

**Navigate to**: `Organization Settings → Secrets and variables → Actions`

1. **PULUMI_ACCESS_TOKEN** - Pulumi Cloud authentication
2. **DIGITALOCEAN_TOKEN** - Digital Ocean API access
3. **GHCR_TOKEN** - Container registry push access
4. **SUBMODULE_TOKEN** - Private submodule access

**Repository access**: Select `All repositories` or specific repos

**Time to set up**: ~10 minutes

**Detailed instructions**: [GITHUB-SECRETS-SETUP.md](./GITHUB-SECRETS-SETUP.md)

## Initial Setup (First Time)

### Step 1: Initialize Pulumi Backend

```bash
# 1. Install Pulumi CLI
curl -fsSL https://get.pulumi.com | sh

# 2. Login to Pulumi Cloud
pulumi login

# 3. Your state will be stored at: https://app.pulumi.com
```

### Step 2: Add Organization Secrets

Add the 4 secrets to **Organization Settings → Secrets and variables → Actions**

(See [GITHUB-SECRETS-SETUP.md](./GITHUB-SECRETS-SETUP.md) for detailed instructions)

### Step 3: (Optional) Configure GitHub Environments

**Only needed if you want manual approval for deployments.**

**Create production environment** (optional):
1. Go to **Settings → Environments → New environment**
2. Name: `production`
3. Add protection rules:
   - ✅ Required reviewers: 1-6 people
   - ✅ Wait timer: 0 minutes (or add delay)
   - ✅ Deployment branches: `main` only

4. Update workflow to use environment:
   ```yaml
   # .github/workflows/production-deploy.yml
   jobs:
     deploy:
       environment: production  # Uncomment this line
   ```

**Skip this step if**: You want fully automated deployments (Flagger provides safety).

### Step 4: Deploy Infrastructure (First Time)

**Option A: Via GitHub Actions (Recommended)**

1. Go to **Actions → Deploy Infrastructure → Run workflow**
2. Select: `production`
3. Click: **Run workflow**
4. Monitor progress in Actions tab
5. Wait ~15-20 minutes for cluster creation

**Option B: Locally (Alternative)**

```bash
cd infra

# Configure
pulumi stack select production
pulumi config set region sfo3
pulumi config set domain eng.in.docutag.app
pulumi config set baseDomain docutag.app
pulumi config set enableFlagger true

# Deploy
export DIGITALOCEAN_TOKEN=your_token
pulumi up
```

### Step 5: Verify Infrastructure

```bash
# Get cluster name
pulumi stack output clusterName

# Connect to cluster
doctl kubernetes cluster kubeconfig save docutag-production

# Verify components
kubectl get nodes
kubectl get pods -n traefik
kubectl get pods -n flagger-system
```

## Ongoing Operations

### Deploy New Version (Automatic)

**Just merge to main**:
```bash
git checkout main
git merge staging
git push
```

**What happens automatically**:
1. Release workflow calculates version (e.g., v1.1.0)
2. Builds and pushes Docker images
3. Creates GitHub release
4. Application deploy workflow deploys to Kubernetes
5. Flagger performs progressive rollout
6. Tests run automatically

**No manual steps required!**

### Deploy Specific Version (Manual)

1. Go to **Actions → Deploy Application to Production**
2. Click **Run workflow**
3. Enter version (e.g., `1.0.0`)
4. Click **Run workflow**

### Update Infrastructure

**Automatic** (when infra code changes):
```bash
# Make changes to infra/**
git add infra/
git commit -m "feat(infra): update cluster configuration"
git push origin main

# Infrastructure workflow runs automatically
```

**Manual**:
1. Go to **Actions → Deploy Infrastructure**
2. Click **Run workflow**
3. Select stack: `production`
4. Click **Run workflow**

## Workflow Details

### Infrastructure Workflow

**Triggers**:
- Manual via GitHub Actions UI
- Automatic on changes to `infra/**`
- Automatic on workflow file changes

**What it does**:
1. Checks out code with submodules
2. Installs Pulumi CLI and Go
3. Configures Pulumi stack (production settings)
4. Runs `pulumi preview` (shows changes)
5. Runs `pulumi up --yes` (applies changes)
6. Verifies cluster access
7. Checks infrastructure components

**Idempotent**: Safe to run multiple times - only applies changes

**Duration**:
- First run: ~15-20 minutes (creates cluster)
- Subsequent runs: ~2-5 minutes (updates only)

### Release Workflow

**Triggers**:
- Automatic on push to `main` branch

**What it does**:
1. Tests version calculation logic
2. Calculates next version from conventional commits
3. Builds Docker images for all 5 services
4. Pushes images to GHCR with version tag
5. Creates git tag (e.g., `v1.1.0`)
6. Creates GitHub release with notes

**Duration**: ~10-15 minutes

### Application Deploy Workflow

**Triggers**:
- Automatic on GitHub release published
- Manual via GitHub Actions UI

**What it does**:
1. Extracts version from release or manual input
2. Connects to DOKS cluster
3. Deploys via Helm with new image version
4. Runs Helm tests for validation
5. Monitors Flagger canary rollout (if enabled)

**Duration**:
- Without Flagger: ~2-3 minutes
- With Flagger canary: ~10-12 minutes
- With Flagger blue-green: ~3-4 minutes

## Monitoring Deployments

### GitHub Actions UI

1. Go to **Actions** tab
2. Select workflow run
3. View logs for each step
4. Check deployment summary

### Kubernetes

```bash
# Watch pods
kubectl get pods -n docutag -w

# Watch Flagger canary
kubectl get canaries -n docutag -w

# Check events
kubectl get events -n docutag --sort-by='.lastTimestamp'

# View logs
kubectl logs -n docutag deployment/docutag-controller -f
```

### Pulumi

```bash
# View stack outputs
pulumi stack output

# View deployment history
pulumi stack history

# View resources
pulumi stack --show-urns
```

## Troubleshooting

### Workflow Fails: "PULUMI_ACCESS_TOKEN not found"

**Solution**: Add secret to GitHub repository settings

### Workflow Fails: "Stack does not exist"

**Solution**:
```bash
# Create stack manually first time
cd infra
pulumi stack init production
```

Or let the infrastructure workflow create it automatically.

### Workflow Fails: "doctl authentication failed"

**Solution**: Verify DIGITALOCEAN_TOKEN is valid
```bash
# Test locally
doctl auth init --access-token YOUR_TOKEN
doctl account get
```

### Deployment Stuck in Progressing

**Check Flagger canary**:
```bash
kubectl describe canary docutag-controller -n docutag
```

**Common causes**:
- Metrics below threshold
- Prometheus unavailable
- Helm tests failing

**Solution**: Check logs and metrics, or manually rollback:
```bash
kubectl flagger rollback docutag-controller -n docutag
```

### Infrastructure Workflow Times Out

**Solution**: Cluster creation takes 15-20 minutes. Increase timeout if needed:
```yaml
jobs:
  infrastructure:
    timeout-minutes: 30  # Increase from default 360
```

## Security Best Practices

1. ✅ **Use environment protection** for production deployments
2. ✅ **Rotate tokens** every 90 days
3. ✅ **Use minimal token scopes** (only what's needed)
4. ✅ **Enable branch protection** on main branch
5. ✅ **Require PR reviews** before merging
6. ✅ **Use OIDC** instead of long-lived tokens (future enhancement)

## Rollback Procedures

### Rollback Application

**Option 1: Deploy previous version**
```bash
# Via GitHub Actions
Actions → Deploy Application to Production → Run workflow
Version: 1.0.0  # Previous version
```

**Option 2: Helm rollback**
```bash
helm rollback docutag -n docutag
```

### Rollback Infrastructure

**Option 1: Pulumi rollback**
```bash
cd infra
pulumi stack history
pulumi stack rollback <update-number>
```

**Option 2: Git revert**
```bash
git revert <commit-hash>
git push origin main
# Infrastructure workflow runs automatically
```

## Cost Optimization

### Free Tier Usage

- ✅ GitHub Actions: 2,000 minutes/month (free)
- ✅ Pulumi Cloud: Free tier (up to 3 stacks)
- ✅ GHCR: Free for public repositories

### Paid Resources

- DOKS cluster: ~$36/month (3 nodes)
- Load balancer: ~$12/month
- Container registry: ~$5/month
- **Total: ~$53/month**

### Reduce Costs

1. Use smaller node sizes for development
2. Scale down during off-hours (manual)
3. Use spot instances (future enhancement)
4. Self-host PostgreSQL (already doing this - saves $240/month)

## Related Documentation

- [Flagger Setup](./FLAGGER-SETUP.md)
- [Blue-Green Deployment](./BLUE-GREEN-DEPLOYMENT.md)
- [Release Process](./RELEASES.md)
- [Infrastructure README](../infra/README.md)

## Support

- GitHub Actions docs: https://docs.github.com/en/actions
- Pulumi docs: https://www.pulumi.com/docs/
- Digital Ocean Kubernetes: https://docs.digitalocean.com/products/kubernetes/
- Flagger docs: https://docs.flagger.app/
