#!/bin/bash

set -e

SERVICES=("api-server" "stt-worker" "llm-worker" "outbox-relay" "infra-migration")
REGISTRY="speech-to-text"
TAG="${1:-latest}"

echo "=== Copying .env files ==="
for svc in "${SERVICES[@]}"; do
    cp .env.example "apps/${svc}/.env"
    echo ">>> apps/${svc}/.env ready"
done
echo ""

echo "=== Building Docker images ==="
echo ""

for svc in "${SERVICES[@]}"; do
    echo ">>> Building ${svc}..."
    docker build \
        --tag "${REGISTRY}-${svc}:${TAG}" \
        --file "apps/${svc}/Dockerfile" \
        .
    echo ">>> ${svc} built successfully"
    echo ""
done