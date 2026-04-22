---
name: go-wire-arch
description: >
  新增 Interface 實作、Repository、Usecase 或任何需要 Wire 注入的元件時，
  必須觸發此 Skill。觸發關鍵字包含：新增 repository、新增 usecase、wire 注入、
  DI、新增 package 實作、新增 interface 綁定、新增 service、新增 handler。
  本 Skill 確保在新增任何可注入元件後，wire.go 與 wire_gen.go 都能正確同步，
  避免 wire 編譯失敗。
---

# Go Wire Clean Architecture — 依賴注入 SOP

## ⚠️ 首要警告：Wire 的致命陷阱

> **每次新增 Interface 實作或 Constructor 後，一定要同步修改 `wire.go`，
> 否則 `wire_gen.go` 不會自動感知，導致 build 失敗或注入錯誤。**

## 專案架構速覽

```
apps/<service>/
├── cmd/main.go                  # 程式進入點，呼叫 di.Initialize...()
└── internal/
    ├── di/
    │   ├── wire.go              # ⭐ 手寫的 Wire 設定（build tag: wireinject）
    │   └── wire_gen.go          # ✅ 自動產生，禁止手動修改
    ├── handler/                 # HTTP Handler，依賴 Usecase 介面
    ├── repository/              # 介面定義（Consumer-defined，ISP 原則）
    └── usecase/                 # 業務邏輯，依賴 Repository 介面

packages/
├── config/                      # AppConfig（統一設定，含 DBConfig、S3Config 等）
├── db/                          # 具體 DB 實作：DAO、HOF（ExecuteWithOutbox）
├── mq/                          # 具體 MQ 實作：RabbitMQPublisher
└── storage/                     # 具體 Storage 實作：S3Storage
```

## 三層職責（必須理解）

| 層級 | 位置 | 職責 |
|------|------|------|
| **High Level** | `apps/.../repository/` | 定義介面（只含該服務需要的方法，ISP 原則） |
| **Middle Level** | `apps/.../di/wire.go` | 撰寫 Provider、用 `wire.Bind` 媒合介面與實作 |
| **Low Level** | `packages/` | 具體 Struct 實作，**不 import 高層** （避免循環依賴） |

---

## 開發 SOP（每次新增功能必須照順序執行）

### Step 1：在 `packages/` 實作具體 Struct

```go
// packages/xxx/my_service.go
package xxx

type MyService struct { /* 欄位 */ }

func NewMyService(cfg Config) (*MyService, error) { ... }

// 實作業務方法（方法簽名必須與 Step 2 的介面對齊）
func (s *MyService) DoSomething(ctx context.Context, ...) error { ... }
```

### Step 2：在 `apps/<service>/internal/repository/` 定義介面

> **ISP 原則：介面只放該 App 真正需要的方法，不複製整個 Struct 的所有方法。**

```go
// apps/<service>/internal/repository/my_repo.go
package repository

import "context"

type MyRepo interface {
    DoSomething(ctx context.Context, ...) error
    // 只列出這個 App 需要用到的方法
}
```

### Step 3：更新 `di/wire.go`（⭐ 最容易被遺忘的步驟）

更新 `ProviderSet`，加入三件事：
1. **Provider 函數**（將 Config 轉換為具體 Struct）
2. **`wire.Bind`**（將具體 Struct 媒合到介面）
3. 將 Provider 函數加入 `wire.NewSet`

```go
//go:build wireinject
// +build wireinject

package di

import (
    "github.com/google/wire"
    "speech.local/apps/<service>/internal/repository"
    "speech.local/packages/xxx"
    // ...
)

var ProviderSet = wire.NewSet(
    // 基礎設施
    NewAppConfig,
    ProvideDB,

    // ⭐ 新增：注入具體實作 + 介面綁定
    ProvideMyService,
    wire.Bind(new(repository.MyRepo), new(*xxx.MyService)),

    // 其他既有的注入...
    repository.NewTaskRepo,
    NewTaskUseCase,
    NewTaskHandler,
)

// ⭐ 新增 Provider 函數
func ProvideMyService(cfg *config.AppConfig) (*xxx.MyService, error) {
    return xxx.NewMyService(cfg.MyServiceConfig)
}

// Initialize 函數（保持不變）
func InitializeXxxDependencies() (*handler.XxxHandler, error) {
    wire.Build(ProviderSet)
    return nil, nil
}
```

### Step 4：執行 Wire 產生 `wire_gen.go`

```bash
# 在 apps/<service> 目錄下執行
cd apps/<service>
go run -mod=mod github.com/google/wire/cmd/wire ./internal/di/
```

> **產生成功後，`wire_gen.go` 會自動更新。禁止手動編輯 `wire_gen.go`。**

### Step 5：驗證編譯

```bash
go build ./...
```

---

## 實際案例對照：加入 S3Storage

以下是本專案 `api-server` 注入 S3Storage 的完整範例，供對照參考。

**`packages/storage/s3_storage.go`（Low Level）**
```go
type S3Storage struct { ... }
func NewS3Storage(cfg S3Config) (*S3Storage, error) { ... }
func (s *S3Storage) GenerateUploadURL(ctx context.Context, objectKey, contentType string) (string, error) { ... }
```

**`apps/api-server/internal/repository/storage_repo.go`（High Level，ISP）**
```go
type StorageRepo interface {
    GenerateUploadURL(ctx context.Context, objectKey, contentType string) (string, error)
    // ⚠️ S3Storage 還有 DownloadToTempFile，但 api-server 不需要，所以不放進來
}
```

**`apps/api-server/internal/di/wire.go`（Middle Level）**
```go
var ProviderSet = wire.NewSet(
    NewAppConfig,
    ProvideDB,
    NewS3Storage,                                                        // Provider
    wire.Bind(new(repository.StorageRepo), new(*storage.S3Storage)),    // 綁定
    repository.NewTaskRepo,
    NewTaskUseCase,
    NewTaskHandler,
)

func NewS3Storage(cfg *config.AppConfig) (*storage.S3Storage, error) {
    return storage.NewS3Storage(cfg.S3Config)
}
```

---

## 常見錯誤與解法

### ❌ 錯誤 1：新增介面後忘記加 `wire.Bind`

**症狀：** `wire: cannot find a provider for repository.MyRepo`

**解法：** 在 `wire.go` 的 `ProviderSet` 加入：
```go
wire.Bind(new(repository.MyRepo), new(*xxx.MyService)),
```

---

### ❌ 錯誤 2：新增 Provider 函數但忘記加入 `wire.NewSet`

**症狀：** `wire: ProvideMyService is not used`，或注入到錯誤的實作。

**解法：** 確認 `ProvideMyService` 有出現在 `wire.NewSet(...)` 的參數列表中。

---

### ❌ 錯誤 3：直接修改 `wire_gen.go`

**症狀：** 下次執行 `wire` 後修改消失，或 `wire` 產生衝突。

**解法：** 所有修改只能在 `wire.go` 進行，`wire_gen.go` 是純產出物。

---

### ❌ 錯誤 4：`wire.go` 缺少 Build Tag

**症狀：** 兩個 `InitializeXxx` 函數衝突，編譯報 `duplicate function`。

**解法：** `wire.go` 檔案開頭必須有：
```go
//go:build wireinject
// +build wireinject
```

---

### ❌ 錯誤 5：Provider 函數參數型別錯誤

**症狀：** `wire: type mismatch` 或 `cannot use ... as type`

**解法：** Provider 函數的輸入參數型別必須是已在 `ProviderSet` 中被提供的型別。例如要用 `*config.AppConfig`，必須先確認 `NewAppConfig` 已在 Set 中。

---

## Config 擴充 SOP（新增設定項目時）

當新的 Struct 需要從 `AppConfig` 讀取設定時：

1. **在 `packages/<xxx>/` 定義 Config Struct：**
```go
type MyConfig struct {
    Host string `mapstructure:"MY_SERVICE_HOST"`
    Port int    `mapstructure:"MY_SERVICE_PORT"`
}
```

2. **將 Config 嵌入 `packages/config/env.go` 的 `AppConfig`：**
```go
type AppConfig struct {
    // ...既有欄位
    MyServiceConfig xxx.MyConfig `mapstructure:",squash"` // squash = 平鋪欄位
}
```

3. **在 `.env.example` 補上對應的環境變數：**
```
MY_SERVICE_HOST=localhost
MY_SERVICE_PORT=8080
```

4. **Provider 函數直接從 AppConfig 取出：**
```go
func ProvideMyService(cfg *config.AppConfig) (*xxx.MyService, error) {
    return xxx.NewMyService(cfg.MyServiceConfig)
}
```

---

## 新增完整功能的 Checklist

完成後請逐項確認：

- [ ] `packages/<xxx>/` 具體 Struct 已實作，方法簽名正確
- [ ] `apps/<service>/internal/repository/<xxx>_repo.go` 介面已定義（ISP 原則）
- [ ] `di/wire.go` 的 `ProviderSet` 已加入新 Provider 函數
- [ ] `di/wire.go` 的 `ProviderSet` 已加入 `wire.Bind(...)`
- [ ] 若有新設定，`AppConfig` 與 `.env.example` 已更新
- [ ] 已執行 `wire ./internal/di/` 更新 `wire_gen.go`
- [ ] `go build ./...` 編譯通過
- [ ] 單元測試使用 Mock（`testify/mock`）而非真實依賴
