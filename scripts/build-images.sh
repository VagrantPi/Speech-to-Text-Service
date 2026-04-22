#!/bin/bash

# =============================================================================
# 自動化建置腳本 - 支援本地開發和 CI/CD
# =============================================================================

set -e

# 顏色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

SERVICES=("api-server" "stt-worker" "llm-worker" "outbox-relay" "infra-migration")
REGISTRY="speech-to-text"
TAG="${1:-latest}"

echo -e "${BLUE}🏗️  Speech-to-Text Service 建置腳本${NC}"
echo ""

# 設置環境（本地開發需要 .env 檔案）
echo -e "${BLUE}📋 步驟 1/2: 環境設置${NC}"
if [ "$CI" != "true" ]; then
    echo "🏠 本地開發環境 - 設置 .env 檔案..."
    ./scripts/setup-env.sh
else
    echo "🤖 CI 環境 - 跳過 .env 複製（由 GitHub Actions 提供）"
fi
echo ""

# 建置 images
echo -e "${BLUE}📋 步驟 2/2: 建置 Docker images${NC}"
echo ""

BUILD_SUCCESS=0
BUILD_FAILED=0

for svc in "${SERVICES[@]}"; do
    echo -e "${YELLOW}🔨 建置 ${svc}...${NC}"
    
    if docker build \
        --tag "${REGISTRY}-${svc}:${TAG}" \
        --file "apps/${svc}/Dockerfile" \
        --progress=plain \
        .; then
        echo -e "${GREEN}   ✅ ${svc} 建置成功${NC}"
        ((BUILD_SUCCESS++))
    else
        echo -e "${RED}   ❌ ${svc} 建置失敗${NC}"
        ((BUILD_FAILED++))
    fi
    echo ""
done

# 總結
echo -e "${BLUE}📊 建置總結${NC}"
echo -e "   ${GREEN}✅ 成功: ${BUILD_SUCCESS}${NC}"
echo -e "   ${RED}❌ 失敗: ${BUILD_FAILED}${NC}"
echo ""

if [ $BUILD_FAILED -gt 0 ]; then
    echo -e "${RED}🚫 建置失敗，請檢查錯誤訊息${NC}"
    exit 1
else
    echo -e "${GREEN}🎉 所有 images 建置完成！${NC}"
    echo ""
    echo -e "${BLUE}📋 建置的 images:${NC}"
    for svc in "${SERVICES[@]}"; do
        echo "   - ${REGISTRY}-${svc}:${TAG}"
    done
fi