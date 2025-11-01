#!/bin/bash
set -e

# Server-side deployment script for DocuTag staging
# Usage: ./deploy-staging.sh

echo "🚀 Deploying DocuTag to staging..."
echo ""

# Pull latest images from GitHub Container Registry
echo "📥 Pulling latest images from ghcr.io/docutag..."
docker compose -f docker-compose.yml -f docker-compose.staging.yml pull

echo ""
echo "🔄 Starting services..."
docker compose -f docker-compose.yml -f docker-compose.staging.yml up -d

echo ""
echo "✅ Staging deployment complete!"
echo ""
echo "📊 Service status:"
docker compose -f docker-compose.yml -f docker-compose.staging.yml ps
