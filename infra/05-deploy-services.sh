#!/bin/bash
set -e

# ============================================================
# Booking Rush - Deploy Services
# ============================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [ -f "$SCRIPT_DIR/.env" ]; then
    source "$SCRIPT_DIR/.env"
fi

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Config
HOST="${HOST:-5.75.233.23}"
SSH_USER="${SSH_USER:-root}"
NAMESPACE="booking-rush"
K8S_DIR="$SCRIPT_DIR/k8s"

# GHCR Config
GHCR_USER="${GHCR_USER:-prohmpiriya}"
GHCR_TOKEN="${GHCR_TOKEN:-}"

print_header() {
    echo -e "\n${BLUE}============================================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}============================================================${NC}\n"
}

print_success() { echo -e "${GREEN}✓ $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠ $1${NC}"; }
print_error() { echo -e "${RED}✗ $1${NC}"; }

ssh_cmd() {
    ssh -o StrictHostKeyChecking=no "$SSH_USER@$HOST" "export KUBECONFIG=/etc/rancher/k3s/k3s.yaml && $@"
}

scp_file() {
    scp -o StrictHostKeyChecking=no "$1" "$SSH_USER@$HOST:$2"
}

# ============================================================
# Setup Functions
# ============================================================
create_namespace() {
    print_header "Creating Namespace"
    scp_file "$K8S_DIR/namespace.yaml" "/tmp/namespace.yaml"
    ssh_cmd "kubectl apply -f /tmp/namespace.yaml"
    print_success "Namespace created"
}

create_ghcr_secret() {
    print_header "Creating GHCR Secret"

    if [ -z "$GHCR_TOKEN" ]; then
        print_warning "GHCR_TOKEN not set, skipping secret creation"
        echo "Set GHCR_TOKEN in .env or environment to pull private images"
        return 0
    fi

    ssh_cmd "kubectl delete secret ghcr-secret -n $NAMESPACE 2>/dev/null || true"
    ssh_cmd "kubectl create secret docker-registry ghcr-secret \
        --docker-server=ghcr.io \
        --docker-username=$GHCR_USER \
        --docker-password=$GHCR_TOKEN \
        -n $NAMESPACE"
    print_success "GHCR secret created"
}

apply_config() {
    print_header "Applying ConfigMap and Secrets"
    scp_file "$K8S_DIR/configmap.yaml" "/tmp/configmap.yaml"
    scp_file "$K8S_DIR/secrets.yaml" "/tmp/secrets.yaml"
    ssh_cmd "kubectl apply -f /tmp/configmap.yaml"
    ssh_cmd "kubectl apply -f /tmp/secrets.yaml"
    print_success "Config applied"
}

# ============================================================
# Deploy Functions
# ============================================================
deploy_service() {
    local name=$1
    local file="$K8S_DIR/$name.yaml"

    if [ ! -f "$file" ]; then
        print_error "File not found: $file"
        return 1
    fi

    echo "Deploying $name..."
    scp_file "$file" "/tmp/$name.yaml"
    if ssh_cmd "kubectl apply -f /tmp/$name.yaml"; then
        print_success "$name deployed"
    else
        print_error "$name deployment failed"
        return 1
    fi
}

deploy_all_services() {
    print_header "Deploying All Services"

    deploy_service "api-gateway"
    deploy_service "auth-service"
    deploy_service "ticket-service"
    deploy_service "booking-service"
    deploy_service "payment-service"
    deploy_service "frontend-web"
    deploy_service "inventory-worker"
    deploy_service "queue-release-worker"
    deploy_service "saga-orchestrator"
    deploy_service "saga-step-worker"
    deploy_service "saga-payment-worker"
    deploy_service "seat-release-worker"

    print_success "All services deployed"
}

deploy_ingress() {
    print_header "Deploying Ingress"
    scp_file "$K8S_DIR/ingress.yaml" "/tmp/ingress.yaml"
    ssh_cmd "kubectl apply -f /tmp/ingress.yaml"
    print_success "Ingress deployed"
}

# ============================================================
# Status Functions
# ============================================================
show_status() {
    print_header "Deployment Status"

    echo "Pods:"
    ssh_cmd "kubectl get pods -n $NAMESPACE -o wide"
    echo ""
    echo "Services:"
    ssh_cmd "kubectl get svc -n $NAMESPACE"
    echo ""
    echo "Ingress:"
    ssh_cmd "kubectl get ingress -n $NAMESPACE" || true
}

wait_for_pods() {
    print_header "Waiting for Pods to be Ready"

    echo "Waiting for deployments..."
    ssh_cmd "kubectl rollout status deployment/api-gateway -n $NAMESPACE --timeout=120s" || true
    ssh_cmd "kubectl rollout status deployment/auth-service -n $NAMESPACE --timeout=120s" || true
    ssh_cmd "kubectl rollout status deployment/ticket-service -n $NAMESPACE --timeout=120s" || true
    ssh_cmd "kubectl rollout status deployment/booking-service -n $NAMESPACE --timeout=120s" || true
    ssh_cmd "kubectl rollout status deployment/payment-service -n $NAMESPACE --timeout=120s" || true
    ssh_cmd "kubectl rollout status deployment/frontend-web -n $NAMESPACE --timeout=120s" || true
    ssh_cmd "kubectl rollout status deployment/inventory-worker -n $NAMESPACE --timeout=120s" || true
    ssh_cmd "kubectl rollout status deployment/queue-release-worker -n $NAMESPACE --timeout=120s" || true
    ssh_cmd "kubectl rollout status deployment/saga-orchestrator -n $NAMESPACE --timeout=120s" || true
    ssh_cmd "kubectl rollout status deployment/saga-step-worker -n $NAMESPACE --timeout=120s" || true
    ssh_cmd "kubectl rollout status deployment/saga-payment-worker -n $NAMESPACE --timeout=120s" || true
    ssh_cmd "kubectl rollout status deployment/seat-release-worker -n $NAMESPACE --timeout=120s" || true

    print_success "All pods ready"
}

# ============================================================
# Menu
# ============================================================
show_menu() {
    echo ""
    echo "Booking Rush - Deploy Services"
    echo "==============================="
    echo "Target: $SSH_USER@$HOST"
    echo "Namespace: $NAMESPACE"
    echo ""
    echo "=== Main ==="
    echo "1) Deploy ALL"
    echo "2) Show status"
    echo ""
    echo "=== Services ==="
    echo "10) Deploy api-gateway"
    echo "11) Deploy auth-service"
    echo "12) Deploy ticket-service"
    echo "13) Deploy booking-service"
    echo "14) Deploy payment-service"
    echo "15) Deploy frontend-web"
    echo ""
    echo "=== Workers ==="
    echo "20) Deploy inventory-worker"
    echo "21) Deploy queue-release-worker"
    echo "22) Deploy saga-orchestrator"
    echo "23) Deploy saga-step-worker"
    echo "24) Deploy saga-payment-worker"
    echo "25) Deploy seat-release-worker"
    echo ""
    echo "=== Infra ==="
    echo "30) Deploy ingress"
    echo ""
    echo "0) Exit"
    echo ""
    read -p "Select an option: " OPTION

    case $OPTION in
        1)
            create_namespace
            create_ghcr_secret
            apply_config
            deploy_all_services
            deploy_ingress
            wait_for_pods
            show_status
            ;;
        2) show_status ;;
        10) deploy_service "api-gateway" ;;
        11) deploy_service "auth-service" ;;
        12) deploy_service "ticket-service" ;;
        13) deploy_service "booking-service" ;;
        14) deploy_service "payment-service" ;;
        15) deploy_service "frontend-web" ;;
        20) deploy_service "inventory-worker" ;;
        21) deploy_service "queue-release-worker" ;;
        22) deploy_service "saga-orchestrator" ;;
        23) deploy_service "saga-step-worker" ;;
        24) deploy_service "saga-payment-worker" ;;
        25) deploy_service "seat-release-worker" ;;
        30) deploy_ingress ;;
        0) echo "Exiting..."; exit 0 ;;
        *) print_error "Invalid option"; show_menu ;;
    esac
}

# ============================================================
# Main
# ============================================================
if [ "$1" == "--all" ]; then
    create_namespace
    create_ghcr_secret
    apply_config
    deploy_all_services
    deploy_ingress
    wait_for_pods
    show_status
elif [ "$1" == "--services" ]; then
    deploy_all_services
    wait_for_pods
elif [ "$1" == "--config" ]; then
    apply_config
elif [ "$1" == "--ingress" ]; then
    deploy_ingress
elif [ "$1" == "--status" ]; then
    show_status
elif [ "$1" == "--help" ] || [ "$1" == "-h" ]; then
    echo "Usage: $0 [OPTION]"
    echo ""
    echo "Deploy services to k3s"
    echo ""
    echo "Options:"
    echo "  --all        Deploy everything (namespace, config, services, ingress)"
    echo "  --services   Deploy services only"
    echo "  --config     Apply configmap and secrets only"
    echo "  --ingress    Deploy ingress only"
    echo "  --status     Show deployment status"
    echo "  --help       Show this help"
    echo ""
    echo "Environment variables:"
    echo "  GHCR_USER    GitHub username (default: prohmpiriya)"
    echo "  GHCR_TOKEN   GitHub token for pulling images"
else
    show_menu
fi
