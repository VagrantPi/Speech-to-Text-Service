#!/bin/bash

# =============================================================================
# Speech-to-Text Service 自動化啟動腳本
# 
# 支援：
# - 本地開發：完整建置 + 啟動
# - CI/CD：僅啟動（images 預先建置）
# =============================================================================

set -e

# 顏色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}🚀 Speech-to-Text Service 啟動腳本${NC}"
echo ""

# 檢查參數
SKIP_BUILD="${1:-false}"
if [ "$1" = "--skip-build" ] || [ "$CI" = "true" ]; then
    SKIP_BUILD="true"
fi

# 步驟 1: 環境設置
echo -e "${BLUE}📋 步驟 1/4: 環境設置${NC}"
if [ "$CI" != "true" ]; then
    echo "🏠 本地開發環境"
    ./scripts/setup-env.sh
else
    echo "🤖 CI 環境 - 使用 GitHub Actions 環境變數"
fi
echo ""

# 步驟 2: 建置 images（本地開發）
if [ "$SKIP_BUILD" = "false" ]; then
    echo -e "${BLUE}📋 步驟 2/4: 建置 Docker images${NC}"
    ./scripts/build-images.sh
    echo ""
else
    echo -e "${BLUE}📋 步驟 2/4: 跳過建置${NC} ${YELLOW}(--skip-build 或 CI 環境)${NC}"
    echo ""
fi

# 步驟 3: 啟動基礎設施
echo -e "${BLUE}📋 步驟 3/4: 啟動基礎設施${NC}"
echo "🔧 啟動 postgres, redis, rabbitmq, minio..."
docker compose -f docker-compose.infra.yml up -d

# 等待基礎設施健康檢查
echo "⏳ 等待基礎設施服務健康..."
sleep 10
echo ""

# 步驟 4: 啟動應用服務
echo -e "${BLUE}📋 步驟 4/4: 啟動應用服務${NC}"
echo "🚀 啟動 migration + 所有應用服務..."
docker compose -f docker-compose.yml -f docker-compose.infra.yml up -d

# 等待服務啟動
echo "⏳ 等待服務完全啟動..."
sleep 5

# 清理 migration 容器（已完成任務）
echo "🧹 清理 migration 容器..."
docker compose -f docker-compose.yml -f docker-compose.infra.yml rm -f infra-migration

# 顯示狀態
echo ""
echo -e "${BLUE}📊 服務狀態${NC}"
docker compose -f docker-compose.yml -f docker-compose.infra.yml ps

echo ""
echo -e "${GREEN}🎉 Speech-to-Text Service 啟動完成！${NC}"
echo ""
echo -e "${BLUE}🔗 服務端點：${NC}"
echo "   🌐 Web UI: http://localhost:8081"
echo "   🌐 API Server: http://localhost:8080"
echo "   🐰 RabbitMQ Management: http://localhost:15672 (guest/guest)"
echo "   💾 MinIO Console: http://localhost:9001 (minioadmin/minioadmin)"
echo ""
echo -e "${YELLOW}💡 提示：${NC}"
echo "   - 使用 './scripts/cleanup.sh' 清理所有資源"
echo "   - 使用 'docker compose logs [service-name]' 查看日誌"
echo "   - 使用 '--skip-build' 參數跳過建置步驟"
