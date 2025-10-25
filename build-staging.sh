#!/bin/bash
set -e

# Build and optionally push staging Docker images for DocuTab
# Usage:
#   ./build-staging.sh              - Build all services for local platform
#   ./build-staging.sh push         - Build multi-platform and push to registry

REGISTRY=${REGISTRY:-ghcr.io/zombar}
PUSH=${1:-}

# Define all services
SERVICES=("textanalyzer" "scraper" "controller" "scheduler" "web")

echo "🔨 Building staging Docker images..."
echo ""

# Function to get Dockerfile path for a service
get_dockerfile() {
    case $1 in
        textanalyzer) echo "./apps/textanalyzer/Dockerfile" ;;
        scraper) echo "./apps/scraper/Dockerfile" ;;
        controller) echo "./apps/controller/Dockerfile" ;;
        scheduler) echo "./apps/scheduler/Dockerfile" ;;
        web) echo "./apps/web/Dockerfile" ;;
    esac
}

# Ensure buildx builder exists for multi-platform builds
if [ "$PUSH" = "push" ]; then
    echo "🔧 Setting up multi-platform builder..."
    docker buildx create --name docutab-builder --use 2>/dev/null || docker buildx use docutab-builder 2>/dev/null || docker buildx use default
    echo ""
fi

# Build all images
if [ "$PUSH" = "push" ]; then
    echo "📦 Building and pushing multi-platform images (amd64, arm64)..."
    echo "   Registry: $REGISTRY"
    echo ""

    for service in "${SERVICES[@]}"; do
        echo "→ Building and pushing $service..."
        dockerfile=$(get_dockerfile "$service")

        if [ "$service" = "web" ]; then
            # Web service needs build args
            docker buildx build \
                --platform linux/amd64,linux/arm64 \
                --build-arg VITE_PUBLIC_URL_BASE= \
                --build-arg VITE_CONTROLLER_API_URL= \
                --build-arg VITE_GRAFANA_URL=http://docutab.honker:3000 \
                -t $REGISTRY/docutab-$service:staging \
                -f $dockerfile \
                . \
                --push
        else
            docker buildx build \
                --platform linux/amd64,linux/arm64 \
                -t $REGISTRY/docutab-$service:staging \
                -f $dockerfile \
                . \
                --push
        fi
        echo "  ✓ $service complete"
        echo ""
    done

    echo "✅ All images built and pushed!"
else
    echo "📦 Building images for local platform..."
    echo ""

    for service in "${SERVICES[@]}"; do
        echo "→ Building $service..."
        dockerfile=$(get_dockerfile "$service")

        if [ "$service" = "web" ]; then
            # Web service needs build args
            docker buildx build \
                --build-arg VITE_PUBLIC_URL_BASE= \
                --build-arg VITE_CONTROLLER_API_URL= \
                --build-arg VITE_GRAFANA_URL=http://docutab.honker:3000 \
                -t docutab-$service:staging \
                -f $dockerfile \
                . \
                --load
        else
            docker buildx build \
                -t docutab-$service:staging \
                -f $dockerfile \
                . \
                --load
        fi
        echo "  ✓ $service complete"
        echo ""
    done

    echo "✅ Build complete!"
fi

echo ""
echo "📋 Images:"
for service in "${SERVICES[@]}"; do
    if [ "$PUSH" = "push" ]; then
        echo "   • $REGISTRY/docutab-$service:staging"
    else
        echo "   • docutab-$service:staging"
    fi
done
