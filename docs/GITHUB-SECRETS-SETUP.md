# GitHub Secrets Setup Guide

Simplified guide for setting up secrets at the **organization level** instead of using GitHub Environments.

## Overview

All secrets can be configured once at the organization level and shared across all repositories and workflows.

## Approach 1: Organization Secrets (Recommended - Simplest)

### Benefits
- ✅ Configure once, use everywhere
- ✅ No need to create environments
- ✅ Centralized secret management
- ✅ Automatic access for all repos in org

### Setup

**Navigate to**: `Organization Settings → Secrets and variables → Actions → New organization secret`

Add these 4 secrets:

---

### 1. PULUMI_ACCESS_TOKEN

**Purpose**: Pulumi Cloud state management

**How to get**:
```bash
1. Sign up: https://app.pulumi.com
2. Settings → Access Tokens → Create token
3. Copy: pul-xxxxx...
```

**Add to GitHub**:
- Name: `PULUMI_ACCESS_TOKEN`
- Value: `pul-xxxxx...`
- Repository access: `All repositories` or `Selected repositories`

---

### 2. DIGITALOCEAN_TOKEN

**Purpose**: Digital Ocean API access

**How to get**:
```bash
1. Visit: https://cloud.digitalocean.com/account/api/tokens
2. Generate New Token
3. Name: "GitHub Actions CI/CD"
4. Permissions: Read + Write
5. Copy token: dop_v1_xxxxx...
```

**Add to GitHub**:
- Name: `DIGITALOCEAN_TOKEN`
- Value: `dop_v1_xxxxx...`
- Repository access: `All repositories`

---

### 3. GHCR_TOKEN

**Purpose**: GitHub Container Registry push access

**How to get**:
```bash
1. Visit: https://github.com/settings/tokens
2. Generate new token (classic)
3. Name: "GHCR Push Token"
4. Scopes:
   ✓ write:packages
   ✓ read:packages
   ✓ delete:packages
5. Generate token
6. Copy: ghp_xxxxx...
```

**Add to GitHub**:
- Name: `GHCR_TOKEN`
- Value: `ghp_xxxxx...`
- Repository access: `All repositories`

---

### 4. SUBMODULE_TOKEN

**Purpose**: Access private submodule repositories

**Option A: Personal Access Token (Simple)**

```bash
1. Visit: https://github.com/settings/tokens
2. Generate new token (classic)
3. Name: "Submodule Access"
4. Scopes:
   ✓ repo (Full control of private repositories)
5. Generate token
6. Copy: ghp_xxxxx...
```

**Option B: GitHub App (Better - Recommended)**

Instead of a PAT, use a GitHub App for better security:

```bash
1. Create GitHub App:
   - Settings → Developer settings → GitHub Apps → New App
   - Name: "DocuTag Submodules"
   - Permissions: Repository → Contents → Read-only
   - Install on organization

2. Generate private key and get App ID

3. Use in workflows:
   - uses: actions/create-github-app-token@v1
     with:
       app-id: ${{ secrets.APP_ID }}
       private-key: ${{ secrets.APP_PRIVATE_KEY }}
```

**For simplicity, use Option A (PAT) for now.**

**Add to GitHub**:
- Name: `SUBMODULE_TOKEN`
- Value: `ghp_xxxxx...`
- Repository access: `All repositories`

**Note**: This token needs access to all submodule repos:
- docutag/controller
- docutag/scraper
- docutag/textanalyzer
- docutag/scheduler
- docutag/web
- docutag/infra

---

## Approach 2: GitHub Environments (If You Want Manual Approval)

If you want **manual approval** for production deployments, use environments:

### Setup

1. **Create production environment**:
   ```
   Repository → Settings → Environments → New environment
   Name: production
   ```

2. **Add protection rules**:
   - ✅ Required reviewers: 1-6 reviewers
   - ✅ Wait timer: 0 minutes (or delay)
   - ✅ Deployment branches: `main` only

3. **Add environment-specific secrets** (optional):
   - PULUMI_ACCESS_TOKEN
   - DIGITALOCEAN_TOKEN

4. **Update workflows** to use environment:
   ```yaml
   jobs:
     deploy:
       runs-on: ubuntu-latest
       environment: production  # Add this line
   ```

### When to Use Environments

**Use environments if**:
- ✅ You want manual approval for production
- ✅ Different secrets per environment
- ✅ Deployment audit trail

**Skip environments if**:
- ✅ Fully automated deployments
- ✅ Same secrets everywhere
- ✅ Flagger provides safety (already have rollback)

---

## Verification

After adding secrets, verify they're accessible:

```bash
# Test by triggering a workflow manually
Actions → Deploy Infrastructure → Run workflow

# Check logs - secrets should be masked with ***
```

## Security Best Practices

1. **Rotate tokens every 90 days**
   ```bash
   # Set calendar reminder to regenerate tokens quarterly
   ```

2. **Use minimal scopes**
   - Only grant permissions needed
   - GHCR: packages only (not full repo)
   - SUBMODULE: read-only if possible

3. **Use GitHub App instead of PAT** (for SUBMODULE_TOKEN)
   - Better security
   - Fine-grained permissions
   - Automatic token rotation

4. **Enable audit logging**
   ```
   Organization Settings → Audit log
   Review secret access patterns
   ```

5. **Use OIDC for cloud providers** (future enhancement)
   ```yaml
   # Instead of long-lived DIGITALOCEAN_TOKEN
   # Use OIDC federation (no secrets needed)
   ```

## Troubleshooting

### Workflow fails: "Secret not found"

**Solution**: Check secret name matches exactly (case-sensitive)

```bash
# ✅ Correct
PULUMI_ACCESS_TOKEN

# ❌ Wrong
pulumi_access_token
Pulumi_Access_Token
```

### Workflow fails: "Resource not accessible"

**Solution**: Check organization secret repository access

```bash
Organization Settings → Secrets → PULUMI_ACCESS_TOKEN
Repository access: All repositories ✓
```

### Submodule checkout fails: "Authentication failed"

**Solution**: SUBMODULE_TOKEN needs repo scope

```bash
# Verify token has access to all submodules
doctl auth init --access-token YOUR_SUBMODULE_TOKEN
gh repo list docutag
```

### GHCR push fails: "403 Forbidden"

**Solution**: GHCR_TOKEN needs write:packages scope

```bash
# Regenerate token with correct scopes
Settings → Tokens → Regenerate
Scopes: ✓ write:packages
```

## Migration from Environments to Org Secrets

If you already have environment secrets, migrate to org secrets:

```bash
# 1. Copy secret values from environments

# 2. Create organization secrets with same names

# 3. Remove environment secrets (optional)

# 4. Remove 'environment:' from workflows (optional)

# 5. Delete unused environments (optional)
```

## Summary

**Minimum setup** (4 organization secrets):
1. PULUMI_ACCESS_TOKEN
2. DIGITALOCEAN_TOKEN
3. GHCR_TOKEN
4. SUBMODULE_TOKEN

**No environments needed** unless you want manual approval.

**Time to set up**: ~10 minutes

**Works for**: All workflows across all repositories
