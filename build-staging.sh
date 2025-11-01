#!/bin/bash
set -e

# Build and optionally push Docker images for DocuTag
# Usage:
#   ./build-staging.sh              - Build all services for local platform
#   ./build-staging.sh push         - Build multi-platform and push with :staging tag
#   ./build-staging.sh push 1.0.0   - Build multi-platform and push with :1.0.0 tag
#
# Environment variables for web build configuration:
#   VITE_PUBLIC_URL_BASE      - Base URL for the application (default: https://docutag.honker)
#   VITE_CONTROLLER_API_URL   - Controller API URL (default: /api for relative routing)
#   VITE_GRAFANA_URL          - Grafana URL (default: https://docutag.honker/grafana)
#   VITE_ASYNQ_URL            - Asynqmon URL (default: https://asynqmon.docutag.honker - subdomain routing)

REGISTRY=${REGISTRY:-ghcr.io/docutag}
PUSH=${1:-}
VERSION=${2:-staging}  # Default to "staging" tag if no version specified

# Vite build configuration with defaults for staging/honker environment
VITE_PUBLIC_URL_BASE=${VITE_PUBLIC_URL_BASE:-https://docutag.honker}
VITE_CONTROLLER_API_URL=${VITE_CONTROLLER_API_URL:-/api}
VITE_GRAFANA_URL=${VITE_GRAFANA_URL:-https://docutag.honker/grafana}
VITE_ASYNQ_URL=${VITE_ASYNQ_URL:-https://asynqmon.docutag.honker}

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
    docker buildx create --name docutag-builder --use 2>/dev/null || docker buildx use docutag-builder 2>/dev/null || docker buildx use default
    echo ""
fi

# Build all images
if [ "$PUSH" = "push" ]; then
    echo "📦 Building and pushing images for amd64..."
    echo "   Registry: $REGISTRY"
    echo "   Version: $VERSION"
    echo ""

    for service in "${SERVICES[@]}"; do
        echo "→ Building and pushing $service:$VERSION..."
        dockerfile=$(get_dockerfile "$service")

        if [ "$service" = "web" ]; then
            # Web service needs build args for Vite environment variables
            docker buildx build \
                --platform linux/amd64 \
                --build-arg VITE_PUBLIC_URL_BASE="$VITE_PUBLIC_URL_BASE" \
                --build-arg VITE_CONTROLLER_API_URL="$VITE_CONTROLLER_API_URL" \
                --build-arg VITE_GRAFANA_URL="$VITE_GRAFANA_URL" \
                --build-arg VITE_ASYNQ_URL="$VITE_ASYNQ_URL" \
                -t $REGISTRY/docutag-$service:$VERSION \
                -f $dockerfile \
                . \
                --push
        else
            docker buildx build \
                --platform linux/amd64 \
                -t $REGISTRY/docutag-$service:$VERSION \
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
    echo "   Version: $VERSION"
    echo ""

    for service in "${SERVICES[@]}"; do
        echo "→ Building $service:$VERSION..."
        dockerfile=$(get_dockerfile "$service")

        if [ "$service" = "web" ]; then
            # Web service needs build args for Vite environment variables
            docker buildx build \
                --build-arg VITE_PUBLIC_URL_BASE="$VITE_PUBLIC_URL_BASE" \
                --build-arg VITE_CONTROLLER_API_URL="$VITE_CONTROLLER_API_URL" \
                --build-arg VITE_GRAFANA_URL="$VITE_GRAFANA_URL" \
                --build-arg VITE_ASYNQ_URL="$VITE_ASYNQ_URL" \
                -t docutag-$service:$VERSION \
                -f $dockerfile \
                . \
                --load
        else
            docker buildx build \
                -t docutag-$service:$VERSION \
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
        echo "   • $REGISTRY/docutag-$service:$VERSION"
    else
        echo "   • docutag-$service:$VERSION"
    fi
done
