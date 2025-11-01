#!/bin/bash
# Deploy Helm chart with automated rollback on test failure
#
# Usage:
#   ./deploy-with-rollback.sh [release-name] [namespace] [values-file]
#
# Examples:
#   ./deploy-with-rollback.sh                                    # Uses defaults
#   ./deploy-with-rollback.sh docutag docutag values-prod.yaml  # Custom config

set -e

# Configuration
RELEASE_NAME="${1:-docutag}"
NAMESPACE="${2:-docutag}"
VALUES_FILE="${3:-./chart/values-production.yaml}"
CHART_PATH="./chart"
TIMEOUT_ROLLOUT="300s"
TIMEOUT_TESTS="180s"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

log_success() {
    echo -e "${GREEN}✓${NC} $1"
}

log_error() {
    echo -e "${RED}✗${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# Validate prerequisites
validate_prerequisites() {
    log_info "Validating prerequisites..."

    # Check if helm is installed
    if ! command -v helm &> /dev/null; then
        log_error "helm is not installed"
        exit 1
    fi

    # Check if kubectl is installed
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed"
        exit 1
    fi

    # Check if chart exists
    if [ ! -d "$CHART_PATH" ]; then
        log_error "Chart not found at $CHART_PATH"
        exit 1
    fi

    # Check if values file exists
    if [ ! -f "$VALUES_FILE" ]; then
        log_error "Values file not found at $VALUES_FILE"
        exit 1
    fi

    # Check if namespace exists
    if ! kubectl get namespace "$NAMESPACE" &> /dev/null; then
        log_error "Namespace $NAMESPACE does not exist"
        exit 1
    fi

    log_success "Prerequisites validated"
}

# Get current release info
get_current_revision() {
    if helm list -n "$NAMESPACE" | grep -q "$RELEASE_NAME"; then
        CURRENT_REV=$(helm history "$RELEASE_NAME" -n "$NAMESPACE" -o json | jq -r '.[-1].revision')
        CURRENT_VERSION=$(helm history "$RELEASE_NAME" -n "$NAMESPACE" -o json | jq -r '.[-1].app_version')
        echo "$CURRENT_REV"
    else
        echo "0"
    fi
}

# Perform the upgrade
perform_upgrade() {
    log_info "Deploying $RELEASE_NAME to namespace $NAMESPACE..."

    if [ "$CURRENT_REV" = "0" ]; then
        log_info "First deployment (no previous revision)"
        helm install "$RELEASE_NAME" "$CHART_PATH" -n "$NAMESPACE" -f "$VALUES_FILE"
    else
        log_info "Upgrading from revision $CURRENT_REV (version $CURRENT_VERSION)"
        helm upgrade "$RELEASE_NAME" "$CHART_PATH" -n "$NAMESPACE" -f "$VALUES_FILE"
    fi

    NEW_REV=$(get_current_revision)
    NEW_VERSION=$(helm list -n "$NAMESPACE" -o json | jq -r ".[] | select(.name==\"$RELEASE_NAME\") | .app_version")

    log_info "New revision: $NEW_REV (version $NEW_VERSION)"
}

# Wait for rollout to complete
wait_for_rollout() {
    log_info "Waiting for rollout to complete (timeout: $TIMEOUT_ROLLOUT)..."

    # Get all deployments in the namespace that belong to this release
    DEPLOYMENTS=$(kubectl get deployments -n "$NAMESPACE" -l "app.kubernetes.io/instance=$RELEASE_NAME" -o jsonpath='{.items[*].metadata.name}')

    if [ -z "$DEPLOYMENTS" ]; then
        log_warning "No deployments found for release $RELEASE_NAME"
        return 0
    fi

    for deployment in $DEPLOYMENTS; do
        log_info "Waiting for deployment: $deployment"
        if kubectl rollout status deployment "$deployment" -n "$NAMESPACE" --timeout="$TIMEOUT_ROLLOUT"; then
            log_success "Deployment $deployment rolled out successfully"
        else
            log_error "Deployment $deployment rollout failed or timed out"
            return 1
        fi
    done

    log_success "All deployments rolled out successfully"
}

# Run Helm tests
run_tests() {
    log_info "Running Helm tests (timeout: $TIMEOUT_TESTS)..."

    # Check if tests exist
    TEST_COUNT=$(kubectl get pods -n "$NAMESPACE" -l "helm.sh/hook=test" --show-labels 2>/dev/null | grep -c "test" || echo "0")

    if [ "$TEST_COUNT" = "0" ]; then
        log_warning "No Helm tests defined in chart"
        log_warning "Skipping test phase (deployment considered successful)"
        return 0
    fi

    log_info "Found $TEST_COUNT test pod(s)"

    # Run tests
    if helm test "$RELEASE_NAME" -n "$NAMESPACE" --logs --timeout="$TIMEOUT_TESTS"; then
        log_success "All tests passed!"
        return 0
    else
        log_error "Tests failed!"
        return 1
    fi
}

# Rollback to previous revision
perform_rollback() {
    log_error "Deployment validation failed!"

    if [ "$CURRENT_REV" = "0" ]; then
        log_error "Cannot rollback (this was the first deployment)"
        log_error "Uninstalling failed release..."
        helm uninstall "$RELEASE_NAME" -n "$NAMESPACE"
        exit 1
    fi

    log_warning "Rolling back to revision $CURRENT_REV (version $CURRENT_VERSION)..."

    if helm rollback "$RELEASE_NAME" "$CURRENT_REV" -n "$NAMESPACE" --wait --timeout="$TIMEOUT_ROLLOUT"; then
        log_success "Rollback to revision $CURRENT_REV completed successfully"

        # Verify rollback
        REVERTED_REV=$(get_current_revision)
        log_info "Current revision after rollback: $REVERTED_REV"

        exit 1
    else
        log_error "Rollback failed!"
        log_error "Manual intervention required"
        exit 2
    fi
}

# Show deployment summary
show_summary() {
    echo ""
    echo "=========================================="
    echo "Deployment Summary"
    echo "=========================================="
    echo "Release:       $RELEASE_NAME"
    echo "Namespace:     $NAMESPACE"
    echo "Chart:         $CHART_PATH"
    echo "Values:        $VALUES_FILE"
    echo "Old Revision:  $CURRENT_REV"
    echo "New Revision:  $NEW_REV"
    echo "Old Version:   $CURRENT_VERSION"
    echo "New Version:   $NEW_VERSION"
    echo "Status:        ${GREEN}SUCCESSFUL${NC}"
    echo "=========================================="
    echo ""

    # Show pod status
    log_info "Current pod status:"
    kubectl get pods -n "$NAMESPACE" -l "app.kubernetes.io/instance=$RELEASE_NAME"
    echo ""

    # Show services
    log_info "Services:"
    kubectl get svc -n "$NAMESPACE" -l "app.kubernetes.io/instance=$RELEASE_NAME"
}

# Main execution
main() {
    echo ""
    echo "=========================================="
    echo "Helm Deploy with Automated Rollback"
    echo "=========================================="
    echo ""

    # Step 1: Validate
    validate_prerequisites

    # Step 2: Get current state
    CURRENT_REV=$(get_current_revision)
    if [ "$CURRENT_REV" != "0" ]; then
        CURRENT_VERSION=$(helm list -n "$NAMESPACE" -o json | jq -r ".[] | select(.name==\"$RELEASE_NAME\") | .app_version")
    else
        CURRENT_VERSION="none"
    fi

    # Step 3: Perform upgrade
    perform_upgrade

    # Step 4: Wait for rollout
    if ! wait_for_rollout; then
        perform_rollback
    fi

    # Step 5: Run tests
    if ! run_tests; then
        perform_rollback
    fi

    # Step 6: Success!
    log_success "Deployment completed successfully!"
    show_summary
}

# Run main with error handling
if main; then
    exit 0
else
    log_error "Deployment failed!"
    exit 1
fi
