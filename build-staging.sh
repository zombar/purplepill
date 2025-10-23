#!/bin/bash
set -e

# Build and optionally push staging Docker images for PurpleTab
# Usage:
#   ./build-staging.sh              - Build all services
#   ./build-staging.sh push         - Build and push all to registry
#   REGISTRY=myregistry ./build-staging.sh push

REGISTRY=${REGISTRY:-ghcr.io/zombar}
PUSH=${1:-}

# Define all services
SERVICES=("textanalyzer" "scraper" "controller" "scheduler" "web")

echo "🔨 Building staging Docker images..."
echo ""

# Build all images
echo "📦 Building all service images..."
docker-compose -f docker-compose.build-staging.yml build

echo ""
echo "✅ Build complete!"

# Push to registry if requested
if [ "$PUSH" = "push" ]; then
    if [ -n "$REGISTRY" ]; then
        echo ""
        echo "📤 Pushing to registry: $REGISTRY"
        for service in "${SERVICES[@]}"; do
            echo "  → Pushing purpletab-$service:staging"
            docker tag purpletab-$service:staging $REGISTRY/purpletab-$service:staging
            docker push $REGISTRY/purpletab-$service:staging
        done
        echo "✅ All images pushed!"
    else
        echo ""
        echo "⚠️  No REGISTRY set. Use: REGISTRY=your-registry ./build-staging.sh push"
        exit 1
    fi
fi

echo ""
echo "📋 Built images:"
for service in "${SERVICES[@]}"; do
    echo "   • purpletab-$service:staging"
    [ -n "$REGISTRY" ] && [ "$PUSH" = "push" ] && echo "     $REGISTRY/purpletab-$service:staging"
done
