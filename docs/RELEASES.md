# Release Process

This document describes how to create and deploy versioned releases of the DocuTag platform.

## Overview

DocuTag uses **semantic versioning** (semver) with automatic version calculation based on **conventional commits**. When staging is merged to main, a release is automatically created with:

- Semantic version tag (e.g., `v1.2.3`)
- Docker images for all services
- GitHub release with auto-generated notes
- Deployment instructions

## Semantic Versioning

We follow [Semantic Versioning 2.0.0](https://semver.org/):

- **MAJOR** version (`1.0.0` → `2.0.0`): Breaking changes
- **MINOR** version (`1.0.0` → `1.1.0`): New features, backward-compatible
- **PATCH** version (`1.0.0` → `1.0.1`): Bug fixes, backward-compatible

## Conventional Commits

The release system parses commit messages to determine version bumps:

### Commit Message Format

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

### Examples

**Patch Release** (Bug Fix):
```bash
git commit -m "fix(controller): resolve race condition in queue processor"
```

**Minor Release** (New Feature):
```bash
git commit -m "feat(web): add dark mode toggle to settings"
```

**Major Release** (Breaking Change):
```bash
git commit -m "feat(controller): redesign API endpoints

BREAKING CHANGE: API endpoints now use /v2/ prefix
```

### Commit Types

- `feat`: New feature (triggers MINOR version bump)
- `fix`: Bug fix (triggers PATCH version bump)
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `test`: Adding or updating tests
- `chore`: Maintenance tasks
- `BREAKING CHANGE`: Breaking change (triggers MAJOR version bump)

## Release Workflow

### Automatic Release (Recommended)

1. **Develop features** on feature branches
2. **Merge to staging** for testing
   ```bash
   git checkout staging
   git merge feature/my-feature
   git push
   ```
3. **Test in staging environment**
4. **Merge staging to main** to trigger release
   ```bash
   git checkout main
   git pull
   git merge staging
   git push  # ← This triggers automatic release!
   ```

### What Happens Automatically

When staging is pushed to main:

1. **.github/workflows/release.yml** runs
2. **Calculates next version** from conventional commits
3. **Builds Docker images** with version tag
   - `ghcr.io/docutag/docutag-controller:1.2.3`
   - `ghcr.io/docutag/docutag-scraper:1.2.3`
   - `ghcr.io/docutag/docutag-textanalyzer:1.2.3`
   - `ghcr.io/docutag/docutag-scheduler:1.2.3`
   - `ghcr.io/docutag/docutag-web:1.2.3`
4. **Pushes images** to GitHub Container Registry
5. **Creates git tag** `v1.2.3`
6. **Creates GitHub release** with auto-generated notes

## Deploying a Release to Production

After a release is created:

### Step 1: View the Release

Go to: https://github.com/docutag/platform/releases

Or use GitHub CLI:
```bash
gh release list
gh release view v1.2.3
```

### Step 2: Update Pulumi Configuration

```bash
cd infra
pulumi config set imageVersion 1.2.3 --stack production
```

### Step 3: Deploy to Production

```bash
pulumi up --stack production
```

This will:
- Pull Docker images with the `:1.2.3` tag
- Update Kubernetes deployments
- Rollout new pods with the versioned images

### Step 4: Verify Deployment

```bash
# Check pod status
kubectl get pods -n docutag

# Verify image versions
kubectl get deployment -n docutag -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.template.spec.containers[0].image}{"\n"}{end}'

# Check application health
curl https://eng.in.docutag.app/health
```

## Rolling Back

If a deployment has issues, rollback to the previous version:

```bash
cd infra

# Find previous version
gh release list

# Update to previous version
pulumi config set imageVersion 1.2.2 --stack production

# Deploy
pulumi up --stack production
```

Kubernetes will automatically rollout the previous version's pods.

## Manual Release (Advanced)

If you need to create a release manually without merging to main:

### Create Tag Manually

```bash
# Determine next version
NEXT_VERSION="1.2.3"

# Create and push tag
git tag -a "v${NEXT_VERSION}" -m "Release v${NEXT_VERSION}"
git push origin "v${NEXT_VERSION}"
```

### Build Images Manually

```bash
./build-staging.sh push ${NEXT_VERSION}
```

### Create Release Manually

```bash
gh release create "v${NEXT_VERSION}" \
  --title "DocuTag Platform v${NEXT_VERSION}" \
  --notes "Manual release"
```

## Version History

View all releases:

```bash
# Using GitHub CLI
gh release list

# Using git tags
git tag -l "v*"

# View specific release
gh release view v1.2.3
```

## Environment Image Tags

Different environments use different image tags:

| Environment | Image Tag | Update Method |
|-------------|-----------|---------------|
| **Development** | `staging` | Automatic on push to `honker` or `main` |
| **Staging** | `staging` | Automatic on push to `honker` or `main` |
| **Production** | `1.2.3` (version) | Manual via Pulumi config |

## Release Checklist

Before merging staging to main:

- [ ] All tests passing in CI
- [ ] Staging environment tested and verified
- [ ] Commit messages follow conventional commits format
- [ ] Breaking changes documented in commit body
- [ ] Migration scripts prepared (if needed)
- [ ] Monitoring dashboards reviewed

After release created:

- [ ] Review GitHub release notes
- [ ] Verify Docker images published
- [ ] Update Pulumi config with new version
- [ ] Deploy to production
- [ ] Verify production deployment
- [ ] Monitor metrics and logs
- [ ] Notify team of release

## Troubleshooting

### Release Workflow Failed

**Check workflow logs:**
```bash
gh run list --workflow=release.yml
gh run view <run-id>
```

**Common issues:**
- Conventional commit parsing errors
- Docker build failures
- Image push authentication issues
- Git tag already exists

### Version Calculation Incorrect

**Verify commit messages:**
```bash
# Get commits since last tag
LAST_TAG=$(git describe --tags --abbrev=0)
git log $LAST_TAG..HEAD --pretty=format:"%s"
```

Ensure commits follow conventional commit format.

### Images Not Published

**Check GHCR permissions:**
```bash
# Login to GHCR
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin

# Pull image to verify
docker pull ghcr.io/docutag/docutag-controller:1.2.3
```

### Deployment Issues

**Check Pulumi state:**
```bash
cd infra
pulumi config get imageVersion --stack production
pulumi stack output --stack production
```

**Check Kubernetes resources:**
```bash
kubectl describe deployment -n docutag docutag-controller
kubectl logs -n docutag -l app.kubernetes.io/component=controller
```

## Best Practices

1. **Use feature branches** for development
2. **Test in staging** before merging to main
3. **Write clear commit messages** following conventional commits
4. **Document breaking changes** in commit body
5. **Pin production to specific versions** (never use `staging` tag in production)
6. **Test rollback procedure** periodically
7. **Monitor after deployments** for at least 24 hours
8. **Keep release notes updated** with migration instructions

## References

- [Semantic Versioning](https://semver.org/)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [GitHub Releases](https://docs.github.com/en/repositories/releasing-projects-on-github)
- [GHCR Documentation](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
