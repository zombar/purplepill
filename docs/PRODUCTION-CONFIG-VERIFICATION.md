# Production Configuration Verification

This document verifies that the production Traefik proxy and Vite configuration correctly point to all required endpoints for the production domain: **https://eng.in.docutag.app**

**Status:** ✅ VERIFIED - All configurations are correct

## Production URLs

The following production URLs have been verified:

| Service | URL | Status |
|---------|-----|--------|
| Web UI | https://eng.in.docutag.app/ | ✅ Configured |
| Controller API | https://eng.in.docutag.app/api | ✅ Configured |
| Grafana | https://eng.in.docutag.app/grafana | ✅ Configured (subpath) |
| Asynqmon | https://asynqmon.eng.in.docutag.app | ✅ Configured (subdomain) |

## Traefik Ingress Configuration

### IngressRoute Resources

File: `chart/templates/ingress/ingressroute.yaml`

All IngressRoute resources are correctly configured:

1. **Web UI IngressRoute**
   - Match: `Host(eng.in.docutag.app) && PathPrefix(/)`
   - Service: `docutag-web:80`
   - TLS: Enabled with Let's Encrypt

2. **Controller API IngressRoute**
   - Match: `Host(eng.in.docutag.app) && PathPrefix(/api)`
   - Service: `docutag-controller:8080`
   - TLS: Enabled with Let's Encrypt

3. **Grafana IngressRoute**
   - Match: `Host(eng.in.docutag.app) && PathPrefix(/grafana)`
   - Service: `docutag-grafana:3000`
   - TLS: Enabled with Let's Encrypt
   - Works correctly with subpath routing

4. **Asynqmon IngressRoute** ⚠️ **IMPORTANT**
   - Match: `Host(asynqmon.eng.in.docutag.app)` (subdomain routing only)
   - Service: `docutag-asynqmon:8080`
   - TLS: Enabled with Let's Encrypt
   - **Note:** Asynqmon does NOT support subpath routing. It must use subdomain routing.

### Helm Values Configuration

File: `chart/values.yaml` and `chart/values-production.yaml`

```yaml
ingress:
  enabled: true
  routes:
    web:
      host: "{{ .Values.global.domain }}"  # eng.in.docutag.app
      path: /
    api:
      host: "{{ .Values.global.domain }}"  # eng.in.docutag.app
      path: /api
    grafana:
      host: "{{ .Values.global.domain }}"  # eng.in.docutag.app
      path: /grafana
    asynqmon:
      host: "asynqmon.{{ .Values.global.domain }}"  # asynqmon.eng.in.docutag.app
      path: /
```

## Vite Build Configuration

### Production Build Environment Variables

The production Docker images are built with the following Vite environment variables:

```bash
VITE_PUBLIC_URL_BASE=https://eng.in.docutag.app
VITE_CONTROLLER_API_URL=/api                        # Relative path
VITE_GRAFANA_URL=https://eng.in.docutag.app/grafana
VITE_ASYNQ_URL=https://asynqmon.eng.in.docutag.app  # Subdomain
```

### Configuration Files

1. **Release Workflow** (`.github/workflows/release.yml`)
   - Correctly passes production Vite URLs to `build-staging.sh`
   - Used when creating versioned releases for production

2. **Build Script** (`build-staging.sh`)
   - Accepts environment variables for Vite configuration
   - Defaults to staging/honker URLs if not specified
   - Production URLs override defaults during release builds

3. **Web Dockerfile** (`apps/web/Dockerfile`)
   - Accepts build args for all Vite environment variables
   - Variables are embedded into the built static assets at build time

### Application Code References

File: `apps/web/src/App.jsx`

```javascript
// Navigation links use the Vite environment variables:
<a href={import.meta.env.VITE_ASYNQ_URL || 'http://localhost:9084'}>
  Asynq Monitor
</a>

<a href={`${import.meta.env.VITE_GRAFANA_URL || 'http://localhost:3000'}/dashboards`}>
  Grafana
</a>
```

File: `apps/web/src/services/api.js`

```javascript
// API base URL uses relative path for Traefik routing
const API_BASE_URL = import.meta.env.VITE_CONTROLLER_API_URL ?? 'http://localhost:9080';
```

## DNS Configuration Requirements

For production deployment, DNS must be configured as follows:

```
# Main domain
eng.in.docutag.app.    A    <load-balancer-ip>

# Asynqmon subdomain (required!)
asynqmon.eng.in.docutag.app.    A    <load-balancer-ip>
```

**Recommended:** Use a wildcard DNS record for flexibility:
```
*.eng.in.docutag.app.    A    <load-balancer-ip>
```

This allows automatic support for any future subdomain services.

## Known Issues & Workarounds

### Asynqmon Subpath Issue

**Issue:** Asynqmon's web UI doesn't properly support subpath deployment (e.g., `/asynqmon`). When deployed at a subpath:
- Asset paths are incorrect
- API calls fail
- UI navigation breaks

**Workaround:** Use subdomain routing instead:
- ❌ `https://eng.in.docutag.app/asynqmon` (doesn't work)
- ✅ `https://asynqmon.eng.in.docutag.app` (works correctly)

**Implementation:**
- IngressRoute matches only `Host()` without `PathPrefix()`
- Helm values use `asynqmon.{{ .Values.global.domain }}`
- Vite config uses full subdomain URL

## Verification Checklist

- [x] Traefik IngressRoute templates use correct host/path matching
- [x] Helm values use correct domain substitution
- [x] Production Vite URLs configured in release.yml workflow
- [x] Staging Vite URLs configured in staging-deploy.yml workflow
- [x] Build script accepts environment-specific URLs
- [x] Web Dockerfile accepts and passes Vite build args
- [x] Application code references Vite environment variables correctly
- [x] NOTES.txt displays correct production URLs after deployment
- [x] Chart README documents subdomain requirement for Asynqmon
- [x] DNS wildcard recommendation documented

## Testing Recommendations

After deploying to production:

1. **Verify DNS Resolution**
   ```bash
   dig eng.in.docutag.app
   dig asynqmon.eng.in.docutag.app
   ```

2. **Test HTTPS Endpoints**
   ```bash
   curl -I https://eng.in.docutag.app/
   curl -I https://eng.in.docutag.app/api/health
   curl -I https://eng.in.docutag.app/grafana
   curl -I https://asynqmon.eng.in.docutag.app/
   ```

3. **Verify TLS Certificates**
   ```bash
   openssl s_client -connect eng.in.docutag.app:443 -servername eng.in.docutag.app
   openssl s_client -connect asynqmon.eng.in.docutag.app:443 -servername asynqmon.eng.in.docutag.app
   ```

4. **Check IngressRoute Status**
   ```bash
   kubectl get ingressroute -n docutag
   kubectl describe ingressroute docutag-asynqmon -n docutag
   ```

5. **Verify Web UI Links**
   - Open https://eng.in.docutag.app/
   - Click "Asynq Monitor" icon in navbar
   - Should navigate to https://asynqmon.eng.in.docutag.app
   - Click "Grafana" icon in navbar
   - Should navigate to https://eng.in.docutag.app/grafana

## Related Documentation

- [Release Process](./RELEASES.md) - Semantic versioning and deployment workflow
- [Helm Chart README](../chart/README.md) - Full chart configuration options
- [CI Setup](../infra/.github/CI-SETUP.md) - Required organization secrets

## Summary

All production configurations have been verified and are correct:

✅ **Traefik IngressRoutes** - Correctly configured for domain and subdomain routing
✅ **Vite Build Config** - Production URLs embedded in release builds
✅ **Application Code** - Uses environment variables correctly
✅ **Documentation** - Updated to reflect subdomain requirement
✅ **DNS Requirements** - Documented for production deployment

**Production is ready to deploy** once DNS records are configured and the release workflow creates the first v1.0.0 images.
