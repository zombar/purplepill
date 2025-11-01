# DocuTag Deployment Strategy

## Overview

DocuTag uses a **hybrid deployment approach** that balances simplicity, cost, and scalability.

## Architecture by Environment

```
┌─────────────────────────────────────────────────────────────┐
│                    Development Flow                          │
└─────────────────────────────────────────────────────────────┘

Local Development (Laptop)
    ├── Tool: docker-compose.yml
    ├── Cost: $0/month
    ├── Speed: Fastest iteration
    └── Purpose: Feature development, debugging
                    ↓
                    ↓ git push
                    ↓
Staging (Server)
    ├── Tool: docker-compose.staging.yml
    ├── Cost: ~$20-40/month (existing server)
    ├── Speed: Fast deployment
    └── Purpose: QA, integration testing
                    ↓
                    ↓ promote to production
                    ↓
Production (Kubernetes - DOKS)
    ├── Tool: Pulumi + Helm Chart
    ├── Cost: ~$53/month
    ├── Speed: Slower but controlled
    └── Purpose: Customer-facing production
        ├── Namespace: docutag (primary)
        ├── Future: docutag-client1
        └── Future: docutag-client2
```

## Why This Approach?

### Local Development: Docker Compose
**Benefits:**
- ✅ Zero cost
- ✅ Instant startup
- ✅ No internet required
- ✅ Easy debugging (attach debugger, view logs)
- ✅ Familiar to all developers

**Trade-offs:**
- ❌ Not production-like
- ❌ Single machine only
- ✅ But that's fine for development!

### Staging: Docker Compose on Server
**Benefits:**
- ✅ Low cost (~$20-40/month for server)
- ✅ Simple deployment (`docker-compose up`)
- ✅ Easy to reset/rebuild
- ✅ Good enough for QA testing
- ✅ No Kubernetes complexity

**Trade-offs:**
- ❌ Not identical to production
- ✅ But close enough for most testing

### Production: Kubernetes (DOKS)
**Benefits:**
- ✅ Auto-scaling
- ✅ High availability
- ✅ Zero-downtime deployments
- ✅ Multi-tenant capable
- ✅ Production-grade monitoring
- ✅ Professional appearance for customers

**Trade-offs:**
- ❌ More complex
- ❌ Costs ~$53/month
- ✅ But worth it for production!

## Cost Comparison

| Environment | Tool | Monthly Cost |
|------------|------|--------------|
| Local Dev | Docker Compose | $0 |
| Staging | Docker Compose (server) | $20-40 |
| Production | Kubernetes (DOKS) | $53 |
| **Total** | | **$73-93/month** |

**vs. All Kubernetes:**
| Environment | Tool | Monthly Cost |
|------------|------|--------------|
| Dev | DOKS | $24 |
| Staging | DOKS | $24 |
| Production | DOKS | $53 |
| **Total** | | **$101/month** |

**Savings: ~$10-30/month with hybrid approach**

## When to Use Each

### Use Docker Compose When:
- 👨‍💻 Developing features locally
- 🧪 Testing on staging server
- 🐛 Debugging issues
- 🚀 Need fast iteration

### Use Kubernetes When:
- 🏢 Deploying to production
- 👥 Serving customers
- 📈 Need auto-scaling
- 🎯 Need HA and reliability
- 🔐 Need multi-tenant isolation

## Migration Path

### Phase 1: Current State
```
Dev → docker-compose (laptop)
Staging → docker-compose.staging.yml (server)
Production → Kubernetes (DOKS)
```

### Phase 2: Move Staging to Kubernetes (Optional)
If you want production parity for staging:
```bash
# Deploy staging to Kubernetes in separate namespace
helm install docutag-staging ./chart \
  -f ./chart/values-staging.yaml \
  --set global.domain=staging.docutag.io \
  --namespace docutag-staging \
  --create-namespace
```

### Phase 3: Multi-Tenant Production
Add more customers to same cluster:
```bash
# Deploy client1
helm install docutag-client1 ./chart \
  -f ./chart/values-production.yaml \
  --set global.domain=client1.docutag.io \
  --namespace docutag-client1 \
  --create-namespace
```

## File Structure

```
docutag/
├── docker-compose.yml              # Local development
├── docker-compose.staging.yml      # Staging server
│
├── chart/                          # Helm chart for Kubernetes
│   ├── values.yaml                 # Base configuration
│   ├── values-dev.yaml             # Dev overrides (if using K8s)
│   ├── values-staging.yaml         # Staging overrides (if using K8s)
│   ├── values-production.yaml      # Production configuration
│   └── templates/                  # Kubernetes resources
│
└── infra/                          # Pulumi infrastructure
    ├── main.go                     # Provisions DOKS cluster
    └── MULTI_TENANT.md             # Guide for adding tenants
```

## Deployment Commands Quick Reference

### Local Development
```bash
docker-compose up
```

### Staging
```bash
# On staging server
docker-compose -f docker-compose.staging.yml up -d
```

### Production (First Time)
```bash
cd infra
pulumi stack select production
pulumi up
```

### Production (Updates)
```bash
cd chart
helm upgrade docutag . -f values-production.yaml -n docutag
```

### Add New Tenant
```bash
helm install docutag-{name} ./chart \
  -f ./chart/values-production.yaml \
  --set global.domain={domain} \
  --namespace docutag-{name} \
  --create-namespace
```

## Best Practices

1. **Develop Locally First**
   - Always test changes locally with docker-compose
   - Only deploy to Kubernetes after local testing

2. **Use Staging for Integration**
   - Test multi-service interactions on staging
   - Staging doesn't need to be Kubernetes

3. **Production is Sacred**
   - Only deploy tested code to production
   - Use Kubernetes for production reliability

4. **Keep It Simple**
   - Don't over-engineer development environments
   - Docker Compose is perfectly fine for dev/staging

5. **Scale When Needed**
   - Start with single production tenant
   - Add namespaces as you get more customers
   - Upgrade nodes if needed

## Summary

**Current Setup:**
- ✅ Docker Compose for local dev (free, fast)
- ✅ Docker Compose for staging (simple, cheap)
- ✅ Kubernetes for production (reliable, scalable)
- ✅ Multi-tenant ready (can add namespaces anytime)

**Total Cost: ~$73-93/month**

This is the sweet spot of simplicity, cost, and production quality!
