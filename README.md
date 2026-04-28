# 🎙️ Speech-to-Text AI 摘要服務

一個基於事件驅動架構的語音轉文字與 AI 摘要平台。使用者上傳音訊檔案後，系統會自動完成語音轉錄，並透過 LLM 即時串流產生重點摘要。

---

## 系統架構

本架構基本參考系統設計作業：[ARCHITECTURE.md](https://hackmd.io/@VagrantPi/SyX_3V1abe)

層級 | 組件名稱 | 職責說明
--- | --- | ---
High Level (Apps) | Usecase / Repository Interface | 定義業務流程與「我需要什麼功能」。介面簽名應保持精簡（ISP）。
Middle Level (DI) | Wire ProviderSet | 負責解決基本型別注入問題，並透過 wire.Bind 進行型別媒合。
Low Level (Packages) | Concrete Structs & DAO | 提供高效能、具體的工具實作。包含 DB 事務機制（HOF）與外部 API 調用。

另外本專案已實作 gitlab ci/cd，並暫時將 docker 部署至 docker hub：https://hub.docker.com/u/vagrantpi

S3 的部分採用 MinIO 作為本地儲存解決方案。基本未來可以替換成 AWS S3 或其他 S3 相容的儲存服務。

### 服務說明

| 服務 | 說明 |
|---|---|
| `web` | 前端網頁介面，提供使用者上傳音訊與查看串流摘要的圖形化介面 |
| `api-server` | HTTP API 服務，負責接收請求、產生上傳 URL、建立任務、串流摘要 |
| `stt-worker` | 語音轉文字 Worker，從 RabbitMQ 消費任務，呼叫 OpenAI Whisper |
| `llm-worker` | LLM 摘要 Worker，呼叫 OpenAI GPT-4o 並透過 Redis PubSub 串流 token |
| `outbox-relay` | Outbox Pattern 中繼服務，將 DB 中的 Outbox 事件轉發至 RabbitMQ |
| `infra-migration` | 一次性執行 DB Schema 遷移與 RabbitMQ 拓撲建置 |

### 技術棧

- **語言 / 框架**：Go 1.26.2、Gin
- **前端**：Nginx、HTML/JS
- **資料庫**：PostgreSQL（GORM）
- **訊息佇列**：RabbitMQ（含 Dead Letter Queue）
- **快取 / PubSub**：Redis
- **物件儲存**：MinIO（S3 相容）
- **AI 服務**：OpenAI Whisper（STT）、GPT-4o（LLM Streaming）
- **依賴注入**：Google Wire
- **可觀測性**：OpenTelemetry（Traces + Metrics）、Zap Logger
- **容器化**：Docker、Docker Compose

---

## 環境需求

- Docker >= 24.x
- Docker Compose >= 2.x
- Go 1.26.2（本地開發）
- OpenAI API Key

---

## 快速啟動

### 1. 複製設定檔

```bash
cp .env.example .env
```

編輯 `.env`，至少填入以下必要欄位：

```env
OPENAI_API_KEY=sk-your-openai-api-key

# 以下為預設本地值，一般不需修改
DB_HOST=localhost
DB_PORT=5432
DB_USER=root
DB_PASSWORD=password
DB_NAME=speech_db
DB_SSLMODE=disable
REDIS_HOST=localhost:6379
MQ_URL=amqp://guest:guest@localhost:5672/
MQ_PREFETCH_COUNT=10
RATE_LIMIT_STT_RPM=50
RATE_LIMIT_LLM_RPM=500
AWS_REGION=us-east-1
AWS_S3_BUCKET=uploads
AWS_ACCESS_KEY=minioadmin
AWS_SECRET_KEY=minioadmin
AWS_ENDPOINT=http://localhost:9000
AWS_PUBLIC_ENDPOINT=http://localhost:9000
EXPIRATION_IN_MINUTES=15
WHISPER_API_URL=http://localhost:9000/asr?task=transcribe&output=json
OLLAMA_API_URL=http://localhost:11434/api/chat
OLLAMA_MODEL=qwen2.5:1.5b
ENV=local
API_PORT=8080
OTEL_SERVICE_NAME=api-server
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
```

### 2. 啟動所有服務（含建置）

```bash
./run.sh
```

或跳過 Docker image 建置（CI 環境或已建置過）：

```bash
./run.sh --skip-build
```

腳本會依序執行：
1. 複製 `.env` 到各服務目錄
2. 建置所有 Docker images
3. 啟動基礎設施（PostgreSQL、Redis、RabbitMQ、MinIO）
4. 執行 DB Migration 與 RabbitMQ Topology 設定
5. 啟動所有應用服務（含 Web 前端）

### 3. 手動分步啟動（本地開發）

```bash
# 啟動基礎設施與 Web 服務
docker compose -f docker-compose.infra.yml up -d
docker compose up web -d

# 執行 DB + MQ 初始化
cd apps/infra-migration && go run cmd/main.go

# 啟動 API Server
cd apps/api-server && go run cmd/main.go

# 啟動 Workers（各自在不同 terminal）
cd apps/stt-worker && go run cmd/main.go
cd apps/llm-worker && go run cmd/main.go
cd apps/outbox-relay && go run cmd/main.go
```

### 服務端點

| 服務 | URL | 預設帳密 / 說明 |
|---|---|---|
| Web UI | http://localhost:8081 | 系統前端介面 |
| API Server | http://localhost:8080 | 後端 API 服務 |
| RabbitMQ Management | http://localhost:15672 | `guest` / `guest` |
| MinIO Console | http://localhost:9001 | `minioadmin` / `minioadmin` |

---

## API 說明

Base URL：`http://localhost:8080/api`

---

### 1. 取得音訊上傳 URL

**`GET /api/upload-url`**

取得 S3 Pre-signed URL，讓客戶端可直接將音訊檔案上傳至 MinIO，無需經過後端伺服器。

**Query Parameters**

| 參數 | 必填 | 說明 | 範例 |
|---|---|---|---|
| `ext` | ✅ | 檔案副檔名 | `mp3`, `wav`, `m4a` |
| `content_type` | ✅ | MIME 類型 | `audio/mpeg`, `audio/wav` |

**支援的副檔名**：`.mp3` `.mp4` `.mpeg` `.mpga` `.m4a` `.wav` `.webm`

**Request**

```bash
curl "http://localhost:8080/api/upload-url?ext=mp3&content_type=audio/mpeg"
```

**Response `200 OK`**

```json
{
  "upload_url": "http://localhost:9000/uploads/audio/550e8400-e29b-41d4-a716-446655440000.mp3?X-Amz-Algorithm=...",
  "s3_key": "uploads/audio/550e8400-e29b-41d4-a716-446655440000.mp3"
}
```

**Response `400 Bad Request`**（缺少參數或不支援的副檔名）

```json
{
  "error": "invalid file extension"
}
```

---

### 2. 確認音訊上傳並建立任務

**`POST /api/tasks/confirm`**

音訊上傳至 S3 後，呼叫此端點建立轉錄任務。系統會寫入 Task 與 Outbox 事件，由 Outbox Relay 轉發至 RabbitMQ 觸發 STT Worker。

**Request Body**

```json
{
  "s3_key": "uploads/audio/550e8400-e29b-41d4-a716-446655440000.mp3"
}
```

| 欄位 | 型別 | 必填 | 說明 |
|---|---|---|---|
| `s3_key` | string | ✅ | 步驟 1 取得的 S3 Key |

**Request**

```bash
curl -X POST http://localhost:8080/api/tasks/confirm \
  -H "Content-Type: application/json" \
  -d '{"s3_key": "uploads/audio/550e8400-e29b-41d4-a716-446655440000.mp3"}'
```

**Response `202 Accepted`**

```json
{
  "task_id": 42,
  "status": "PENDING"
}
```

**Response `400 Bad Request`**

```json
{
  "error": "invalid request body: s3_key is required"
}
```

---

### 3. 查詢任務詳情

**`GET /api/tasks/:id`**

查詢指定任務的當前狀態，包含逐字稿與摘要（處理完成後才有值）。

**Path Parameters**

| 參數 | 說明 |
|---|---|
| `id` | 任務 ID（由步驟 2 取得） |

**Request**

```bash
curl "http://localhost:8080/api/tasks/42"
```

**Response `200 OK`**

```json
{
  "ID": 42,
  "Status": "COMPLETED",
  "S3Key": "uploads/audio/550e8400-e29b-41d4-a716-446655440000.mp3",
  "Transcript": "今天會議討論了三個主要議題，分別是 Q3 營收回顧、新產品上市時程，以及組織調整方向...",
  "Summary": "**會議重點摘要**\n\n1. Q3 營收較去年同期成長 15%...",
  "CreatedAt": "2025-04-23T10:00:00Z",
  "UpdatedAt": "2025-04-23T10:02:30Z"
}
```

**任務狀態說明**

| Status | 說明 |
|---|---|
| `CREATED` | 任務剛建立，等待 Outbox Relay 轉發 |
| `PENDING` | 已進入 RabbitMQ，等待 STT Worker 處理 |
| `PROCESSING` | STT 完成，LLM Worker 正在產生摘要 |
| `COMPLETED` | 全部完成，逐字稿與摘要均已寫入 |

**Response `404 Not Found`**

```json
{
  "error": "task not found"
}
```

---

### 4. 串流摘要（SSE）

**`GET /api/tasks/:id/stream`**

透過 Server-Sent Events（SSE）即時接收 LLM 產生的摘要 token 串流。建議在呼叫 `POST /confirm` 後立即建立此連線，以獲得最完整的串流體驗。

**Path Parameters**

| 參數 | 說明 |
|---|---|
| `id` | 任務 ID |

**Request**

```bash
curl -N "http://localhost:8080/api/tasks/42/stream"
```

或在 JavaScript 中：

```javascript
const eventSource = new EventSource('http://localhost:8080/api/tasks/42/stream');

eventSource.onmessage = (event) => {
  const data = JSON.parse(event.data);

  if (data.token === '[DONE]') {
    eventSource.close();
    return;
  }

  // 逐 token 累加顯示
  summaryEl.innerText += data.token;
};
```

**Response（SSE 串流）**

```
data: {"token": "**"}

data: {"token": "會議"}

data: {"token": "重點"}

data: {"token": "摘要"}

data: {"token": "**\n\n"}

data: {"token": "1."}

...

data: {"token": "[DONE]"}
```

收到 `[DONE]` token 表示摘要串流結束。

---

## 運行環境模式 (ENV)

系統支援透過 `ENV` 環境變數切換不同的運行模式，以適應開發、除錯與正式上線的需求：

### 1. 本地開發模式 (`ENV=local`)
**系統預設模式**。會完整執行所有的外部依賴服務，適合開發者在本機進行端到端的功能驗證。
- **行為**：真實連線至本地 Docker 部署的基礎設施（MinIO、RabbitMQ、Redis、PostgreSQL）。
- **AI 服務**：預設串接本地的 Whisper 與 Ollama API，或者可透過環境變數切換為真實的 OpenAI 服務。

#### 本地內部服務端點 (Internal Endpoints)
這些端點主要供 Worker 內部呼叫使用。在本地開發時，您也可以直接呼叫它們來驗證 AI 服務是否正常運作：

| 服務 | 本地 URL | 說明 |
|---|---|---|
| **Whisper (STT)** | `http://localhost:9000/asr` | 提供語音轉文字 API (參數可帶 `?task=transcribe&output=json`) |
| **Ollama (LLM)** | `http://localhost:11434/api/chat` | 提供 LLM 串流對話 API (預設使用 `qwen2.5:1.5b` 模型) |

*(註：若修改了 `docker-compose.infra.yml` 中的對應 Port，請同步更新 `.env` 中的 `WHISPER_API_URL` 與 `OLLAMA_API_URL`)*

---

### 2. 除錯與測試模式 (`ENV=mock`)
專為前端開發與快速測試設計，略過耗時的實體檔案上傳與音訊處理流程。
- **環境設定**：`ENV=mock`
- **運作方式**：
  1. **取得上傳 URL**: API Server 返回 mock URL（包含 `?mock=true`）。
  2. **前端上傳**: 前端偵測到 mock URL 後，改呼叫 `/api/mock/upload/:s3key`。
  3. **Mock 上傳 endpoint**: 模擬上傳成功並直接建立任務。
  4. **STT Worker**: 跳過 S3 下載，直接使用 `/tmp/mock-audio.wav`。

#### Mock 專屬 API 端點

**`POST /api/mock/upload/:s3key`**

Mock 上傳端點，模擬檔案已上傳並建立任務。

| 參數 | 說明 |
|---|---|
| `:s3key` | S3 Key（從 `/api/upload-url` 取得） |

**Request**

```bash
curl -X POST "http://localhost:8080/api/mock/upload/uploads/audio/550e8400-e29b-41d4-a716-446655440000.mp3"
```

**Response**

```json
{
  "status": "uploaded",
  "task_id": 42
}
```

---

### 3. 正式環境模式 (`ENV=production`)
用於正式伺服器部署與維運環境。
- **行為**：API 框架（如 Gin）會自動切換至 Release 模式，關閉冗餘的 Debug Log 輸出以提升整體效能。
- **基礎設施**：通常會搭配正式環境的雲端資源（如真實的 AWS S3 儲存體、代管的 RDS 與 Message Queue）。

---

## 完整使用流程

```bash
# 步驟 1：取得上傳 URL
UPLOAD_INFO=$(curl -s "http://localhost:8080/api/upload-url?ext=mp3&content_type=audio/mpeg")
UPLOAD_URL=$(echo $UPLOAD_INFO | jq -r '.upload_url')
S3_KEY=$(echo $UPLOAD_INFO | jq -r '.s3_key')

# 步驟 2：直接上傳音訊至 S3
curl -X PUT "$UPLOAD_URL" \
  -H "Content-Type: audio/mpeg" \
  --data-binary @your-audio.mp3

# 步驟 3：建立任務
TASK=$(curl -s -X POST http://localhost:8080/api/tasks/confirm \
  -H "Content-Type: application/json" \
  -d "{\"s3_key\": \"$S3_KEY\"}")
TASK_ID=$(echo $TASK | jq -r '.task_id')

# 步驟 4：監聽摘要串流
curl -N "http://localhost:8080/api/tasks/$TASK_ID/stream"
```

---

## 開發指南

### 執行測試

```bash
# 全部
go test ./...

# 指定服務
go test ./apps/api-server/...
go test ./apps/outbox-relay/...
go test ./apps/stt-worker/...
```

### 更新 Wire 依賴注入

修改任何 `wire.go` 後需重新產生：

```bash
./scripts/sync-di.sh

# 或針對單一服務
cd apps/api-server && wire ./internal/di/
```

### 清理所有資源

```bash
./scripts/cleanup.sh
```

---

## 環境變數總覽

| 變數 | 說明 | 預設值 |
|---|---|---|
| `DB_HOST` | PostgreSQL 主機 | `localhost` |
| `DB_PORT` | PostgreSQL 埠號 | `5432` |
| `DB_USER` | DB 帳號 | `root` |
| `DB_PASSWORD` | DB 密碼 | `password` |
| `DB_NAME` | DB 名稱 | `speech_db` |
| `DB_SSLMODE` | SSL 模式 | `disable` |
| `REDIS_HOST` | Redis 位址 | `localhost:6379` |
| `REDIS_PASSWORD` | Redis 密碼 | （空） |
| `MQ_URL` | RabbitMQ AMQP URL | `amqp://guest:guest@localhost:5672/` |
| `MQ_PREFETCH_COUNT` | RabbitMQ QoS prefetch count，限制單 Worker 同時處理任務數 | `10` |
| `RATE_LIMIT_STT_RPM` | STT (Whisper) 每分鐘最大請求數 | `50` |
| `RATE_LIMIT_LLM_RPM` | LLM (GPT-4o) 每分鐘最大請求數 | `500` |
| `AWS_REGION` | S3 Region | `us-east-1` |
| `AWS_S3_BUCKET` | S3 Bucket 名稱 | `uploads` |
| `AWS_ACCESS_KEY` | S3 Access Key | `minioadmin` |
| `AWS_SECRET_KEY` | S3 Secret Key | `minioadmin` |
| `AWS_ENDPOINT` | S3 Endpoint（MinIO API） | `http://localhost:9000` |
| `AWS_PUBLIC_ENDPOINT` | S3 公開端點（供 Pre-signed URL 使用） | `http://localhost:9000` |
| `EXPIRATION_IN_MINUTES` | Pre-signed URL 效期（分鐘） | `15` |
| `WHISPER_API_URL` | Whisper STT API 端點（本地模式） | `http://localhost:9000/asr?task=transcribe&output=json` |
| `OLLAMA_API_URL` | Ollama LLM API 端點（本地模式） | `http://localhost:11434/api/chat` |
| `OLLAMA_MODEL` | Ollama 模型名稱（本地模式） | `qwen2.5:1.5b` |
| `OPENAI_API_KEY` | OpenAI API 金鑰 | **必填** |
| `ENV` | 環境模式（`mock`=除錯模式，`local`=本地開發，`production`=正式環境） | `local` |
| `API_PORT` | API Server 監聽埠號 | `8080` |
| `OTEL_SERVICE_NAME` | OpenTelemetry 服務名稱 | `api-server` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP 收集器位址 | `http://localhost:4317` |

---

## DB Schema

### 1. tasks 表

| 欄位 | 類型 | 說明 |
|------|------|------|
| id | uint (PK, autoIncrement) | 主鍵 |
| status | varchar(50), indexed | 任務狀態 |
| s3_key | varchar(255) | S3 檔案路徑 |
| transcript | text | 語音轉文字結果 (STT 完成後寫入) |
| summary | text | AI 摘要結果 (LLM 完成後寫入) |
| created_at | timestamp | 建立時間 |
| updated_at | timestamp | 更新時間 |

### 2. outbox_events 表 (發件匣)

| 欄位 | 類型 | 說明 |
|------|------|------|
| id | uint (PK, autoIncrement) | 主鍵 |
| aggregate_type_id | uint16 | 業務實體類型 (1=Task) |
| aggregate_id | uint | 關聯的 Task ID |
| topic | varchar(100) | 事件主題 (如 `task.completed`) |
| payload | jsonb | 事件資料 |
| status | varchar(20), indexed | PENDING/PROCESSED |
| retry_count | int | 重試次數 |
| error_reason | text | 錯誤訊息 |
| created_at | timestamp | 建立時間 |

#### 索引

- `idx_outbox_aggregate`: (aggregate_type_id, aggregate_id) 複合索引
- `idx_outbox_pending`: partial index on `where status='PENDING'`


