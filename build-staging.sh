#!/bin/bash
set -e

# Build and optionally push staging Docker images for PurpleTab
# Usage:
#   ./build-staging.sh              - Build all services for local platform
#   ./build-staging.sh push         - Build multi-platform and push to registry

REGISTRY=${REGISTRY:-ghcr.io/zombar}
PUSH=${1:-}

# Define all services
SERVICES=("textanalyzer" "scraper" "controller" "scheduler" "web")

echo "🔨 Building staging Docker images..."
echo ""

# Function to get context path for a service
get_context() {
    case $1 in
        textanalyzer) echo "./apps/textanalyzer" ;;
        scraper) echo "./apps/scraper" ;;
        controller) echo "./apps/controller" ;;
        scheduler) echo "./apps/scheduler" ;;
        web) echo "./apps/web" ;;
    esac
}

# Ensure buildx builder exists for multi-platform builds
if [ "$PUSH" = "push" ]; then
    echo "🔧 Setting up multi-platform builder..."
    docker buildx create --name purpletab-builder --use 2>/dev/null || docker buildx use purpletab-builder 2>/dev/null || docker buildx use default
    echo ""
fi

# Build all images
if [ "$PUSH" = "push" ]; then
    echo "📦 Building and pushing multi-platform images (amd64, arm64)..."
    echo "   Registry: $REGISTRY"
    echo ""

    for service in "${SERVICES[@]}"; do
        echo "→ Building and pushing $service..."
        context=$(get_context "$service")

        if [ "$service" = "web" ]; then
            # Web service needs build args
            docker buildx build \
                --platform linux/amd64,linux/arm64 \
                --build-arg VITE_PUBLIC_URL_BASE=/ \
                --build-arg VITE_CONTROLLER_API_URL= \
                -t $REGISTRY/purpletab-$service:staging \
                -f $context/Dockerfile \
                $context \
                --push
        else
            docker buildx build \
                --platform linux/amd64,linux/arm64 \
                -t $REGISTRY/purpletab-$service:staging \
                -f $context/Dockerfile \
                $context \
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
        context=$(get_context "$service")

        if [ "$service" = "web" ]; then
            # Web service needs build args
            docker buildx build \
                --build-arg VITE_PUBLIC_URL_BASE=/ \
                --build-arg VITE_CONTROLLER_API_URL= \
                -t purpletab-$service:staging \
                -f $context/Dockerfile \
                $context \
                --load
        else
            docker buildx build \
                -t purpletab-$service:staging \
                -f $context/Dockerfile \
                $context \
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
        echo "   • $REGISTRY/purpletab-$service:staging"
    else
        echo "   • purpletab-$service:staging"
    fi
done
