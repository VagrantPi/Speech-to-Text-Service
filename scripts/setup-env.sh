#!/bin/bash

# =============================================================================
# 環境設置腳本 - 支援本地開發和 CI/CD
# 
# 用途：
# - 本地開發：複製 .env.example 到各服務目錄
# - CI/CD：檢查環境變數是否設置
# =============================================================================

set -e

# 顏色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

SERVICES=("api-server" "stt-worker" "llm-worker" "outbox-relay" "infra-migration")

echo -e "${BLUE}🔧 環境設置腳本${NC}"
echo ""

# 檢查是否為 CI 環境
if [ "$CI" = "true" ]; then
    echo -e "${GREEN}🤖 檢測到 CI 環境${NC}"
    echo "✅ CI 環境中，環境變數應由 GitHub Actions 提供"
    echo "✅ Docker images 不包含 .env 檔案（安全最佳實踐）"
    echo ""
    echo "📋 必要的環境變數檢查："
    
    REQUIRED_VARS=(
        "DB_HOST"
        "DB_PORT" 
        "DB_USER"
        "DB_PASSWORD"
        "DB_NAME"
        "REDIS_HOST"
        "MQ_URL"
        "AWS_ENDPOINT"
    )
    
    MISSING_VARS=()
    for var in "${REQUIRED_VARS[@]}"; do
        if [ -z "${!var}" ]; then
            MISSING_VARS+=("$var")
        else
            echo "   ✅ $var"
        fi
    done
    
    if [ ${#MISSING_VARS[@]} -gt 0 ]; then
        echo -e "${RED}❌ 缺少環境變數：${MISSING_VARS[*]}${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}✅ 所有必要環境變數已設置${NC}"
    
else
    echo -e "${GREEN}🏠 本地開發環境${NC}"
    echo "📝 複製 .env.example 到各服務目錄..."
    
    # 檢查 .env.example 是否存在
    if [ ! -f ".env.example" ]; then
        echo -e "${RED}❌ .env.example 檔案不存在${NC}"
        exit 1
    fi
    
    for svc in "${SERVICES[@]}"; do
        ENV_DIR="apps/${svc}"
        ENV_FILE="${ENV_DIR}/.env"
        
        # 創建目錄（如果不存在）
        mkdir -p "$ENV_DIR"
        
        # 複製 .env.example
        cp .env.example "$ENV_FILE"
        echo -e "   ${GREEN}✅${NC} ${ENV_FILE}"
    done
    
    echo ""
    echo -e "${YELLOW}💡 提示：${NC}"
    echo "   - 請根據需要修改各服務的 .env 檔案"
    echo "   - 敏感資訊（如 API keys）請勿提交到版本控制"
    echo "   - 生產環境應使用環境變數或 secrets management"
fi

echo ""
echo -e "${GREEN}🎉 環境設置完成！${NC}"
