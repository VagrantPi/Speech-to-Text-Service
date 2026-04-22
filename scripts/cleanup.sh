#!/bin/bash

# =============================================================================
# Speech-to-Text Service 清理腳本
# 
# 此腳本會刪除所有相關的 Docker images、containers、networks 和 volumes
# 用於測試 run.sh 腳本是否能從零開始正常運作
# =============================================================================

set -e

echo "🧹 開始清理 Speech-to-Text Service 相關資源..."

# 顏色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 確認提示
echo -e "${YELLOW}⚠️  警告：此腳本將刪除以下資源：${NC}"
echo "   - 所有 Speech-to-Text 相關的 Docker containers"
echo "   - 所有 Speech-to-Text 相關的 Docker images"
echo "   - speech-network 網路"
echo "   - pgdata 和 miniodata volumes"
echo ""
read -p "確定要繼續嗎？(y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "❌ 操作已取消"
    exit 1
fi

echo ""
echo "🔄 步驟 1/5: 停止並移除所有 containers..."
docker compose -f docker-compose.yml -f docker-compose.infra.yml down --remove-orphans 2>/dev/null || true

echo ""
echo "🔄 步驟 2/5: 移除相關 images..."
# 移除專案相關的 images
IMAGES=(
    "speech-to-text-api-server"
    "speech-to-text-stt-worker"
    "speech-to-text-llm-worker"
    "speech-to-text-outbox-relay"
    "speech-to-text-infra-migration"
)

for image in "${IMAGES[@]}"; do
    if docker images -q "$image" | grep -q .; then
        echo "   🗑️  刪除 image: $image"
        docker rmi "$image" 2>/dev/null || true
    fi
done

# 移除 dangling images
echo "   🗑️  刪除 dangling images..."
docker image prune -f 2>/dev/null || true

echo ""
echo "🔄 步驟 3/5: 移除相關 volumes..."
VOLUMES=(
    "speech-to-text-service_pgdata"
    "speech-to-text-service_miniodata"
    "pgdata"
    "miniodata"
)

for volume in "${VOLUMES[@]}"; do
    if docker volume ls -q | grep -q "^$volume$"; then
        echo "   🗑️  刪除 volume: $volume"
        docker volume rm "$volume" 2>/dev/null || true
    fi
done

echo ""
echo "🔄 步驟 4/5: 移除相關 networks..."
NETWORKS=(
    "speech-network"
)

for network in "${NETWORKS[@]}"; do
    if docker network ls -q | grep -q "$network"; then
        echo "   🗑️  刪除 network: $network"
        docker network rm "$network" 2>/dev/null || true
    fi
done

echo ""
echo "🔄 步驟 5/5: 最終清理..."
# 清理任何剩餘的相關 containers
CONTAINERS=$(docker ps -a --filter "name=speech-to-text-service" --format "{{.ID}}" 2>/dev/null || true)
if [ ! -z "$CONTAINERS" ]; then
    echo "   🗑️  移除剩餘 containers..."
    docker rm -f $CONTAINERS 2>/dev/null || true
fi

# 清理未使用的資源
echo "   🗑️  清理未使用的 Docker 資源..."
docker system prune -f 2>/dev/null || true

echo ""
echo -e "${GREEN}✅ 清理完成！${NC}"
echo ""
echo "📊 當前狀態："
echo "   Images: $(docker images --filter "reference=speech-to-text*" --format "table {{.Repository}}:{{.Tag}}" | wc -l | tr -d ' ') 個相關 images"
echo "   Containers: $(docker ps -a --filter "name=speech-to-text-service" --format "{{.Names}}" | wc -l | tr -d ' ') 個相關 containers"
echo "   Volumes: $(docker volume ls --filter "name=speech-to-text-service" --format "{{.Name}}" | wc -l | tr -d ' ') 個相關 volumes"
echo "   Networks: $(docker network ls --filter "name=speech-to-text-service" --format "{{.Name}}" | wc -l | tr -d ' ') 個相關 networks"
echo ""
echo "🚀 現在可以測試你的 run.sh 腳本了！"
