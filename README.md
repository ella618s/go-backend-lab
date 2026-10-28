# go-backend-lab

[![CI](https://github.com/ella618s/go-backend-lab/actions/workflows/ci.yml/badge.svg)](https://github.com/ella618s/go-backend-lab/actions/workflows/ci.yml)


使用 Go（Gin、GORM）實作的 RESTful API 練習專案：使用者註冊與登入、JWT 認證、任務管理，並確保每位使用者只能存取自己的資料。

> > 這是學習用專案。預設用 Docker Compose 啟動 PostgreSQL；沒有設定 `DATABASE_URL` 時，會改用本機 SQLite，方便直接 `go run`。

## 技術

- Go、Gin（HTTP 路由與輸入驗證）
- GORM + PostgreSQL（Docker Compose）／SQLite（本機開發與測試）
- JWT（`golang-jwt/jwt/v5`）、bcrypt 密碼雜湊
- `net/http/httptest` 的 API 測試
- Dockerfile（多階段建置）
- WebSocket 聊天室（gorilla/websocket，Hub 模式，goroutine + channel）
- Redis 快取（Cache-Aside）

## 專案結構

```
.
├── main.go              程式進入點：讀取環境變數、初始化資料庫、啟動伺服器
├── router/              路由設定（main 與測試共用）
├── handlers/            處理 HTTP 請求（註冊、登入、任務、使用者）
├── middlewares/         AuthMiddleware：驗證 JWT 並把 user_id 放進 Context
├── models/              資料表結構（User、Task）
└── database/            資料庫連線與自動遷移
```

## 快速開始

需要 Go（版本見 `go.mod`）。

```bash
# 1. 設定環境變數（JWT_SECRET 至少 32 字元）
export JWT_SECRET=$(openssl rand -hex 32)

# 2. 啟動
go run .
# 伺服器預設在 http://localhost:8080
```

環境變數：

| 名稱 | 必填 | 說明 |
| --- | --- | --- |
| `JWT_SECRET` | 是 | 簽發與驗證 Token 的密鑰，至少 32 字元，沒設定程式不會啟動 |
| `DATABASE_URL` | 否 | PostgreSQL 連線字串；有設定就用 PostgreSQL，沒設定則用 SQLite |
| `POSTGRES_PASSWORD` | 用 Compose 時必填 | PostgreSQL 密碼，由 Compose 讀取 `.env` |
| `DB_PATH` | 否 | SQLite 檔案路徑，預設 `app.db` |
| `PORT` | 否 | 監聽埠，預設 `8080` |
| `REDIS_ADDR` | 否 | Redis 位址（例如 `redis:6379`）；沒設定或連不上就不使用快取 |

### 用 Docker Compose 執行（PostgreSQL）

```bash
# 1. 建立 .env（包含 JWT_SECRET 與資料庫密碼，不會提交到 Git）
printf "JWT_SECRET=%s\nPOSTGRES_PASSWORD=%s\n" "$(openssl rand -hex 32)" "$(openssl rand -hex 16)" > .env

# 2. 啟動 app 與 PostgreSQL
docker compose up -d --build

# 3. 查看狀態與紀錄
docker compose ps
docker compose logs app

# 停止（資料保留在 volume）
docker compose down
```

資料庫資料存在 Docker volume `pgdata`，容器重啟不會消失。電腦睡眠或 Docker Desktop 關閉後，再執行 `docker compose up -d` 即可恢復。

### 執行測試

```bash
go test ./...
```

> `go-sqlite3` 需要 cgo，本機要有 C 編譯器（macOS 安裝 Xcode Command Line Tools 即可）。

## API

除了註冊與登入，所有路由都需要在標頭帶 `Authorization: Bearer <token>`。

| 方法 | 路徑 | 說明 |
| --- | --- | --- |
| POST | `/api/register` | 註冊（`name`、`email`、`password` 至少 8 字元） |
| POST | `/api/login` | 登入，取得有效 2 小時的 Token |
| GET | `/api/me` | 取得自己的資料與任務（使用 Preload 載入一對多關聯） |
| POST | `/api/tasks` | 新增任務（`title` 必填，`status` 可為 `pending` 或 `done`） |
| GET | `/api/tasks` | 列出自己的任務，可用 `?status=done` 篩選 |
| PUT | `/api/tasks/:id` | 修改自己的任務 |
| DELETE | `/api/tasks/:id` | 刪除自己的任務 |
| POST | `/api/heavy-task` | 示範 goroutine 背景任務，同時最多 5 個 |
| GET | `/chat` | 聊天室測試頁面 |
| GET | `/api/ws` | WebSocket 聊天室，Token 以 `?token=` 帶入 |

### 範例

```bash
# 註冊
curl -X POST localhost:8080/api/register \
  -H 'Content-Type: application/json' \
  -d '{"name":"Alice","email":"alice@example.com","password":"password123"}'

# 登入，取得 token
curl -X POST localhost:8080/api/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"alice@example.com","password":"password123"}'

# 新增任務
curl -X POST localhost:8080/api/tasks \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"title":"寫 README"}'

# 列出任務
curl localhost:8080/api/tasks -H "Authorization: Bearer $TOKEN"
```

## 設計重點

- **密碼**：用 bcrypt 雜湊後才存入資料庫，`User.Password` 標記 `json:"-"`，任何回應都不會輸出。
- **JWT**：只放 `user_id` 與 `email`，設定過期時間；驗證時只接受 HS256，沒有過期時間的 Token 會被拒絕。密鑰由環境變數提供。
- **資料隔離**：任務的所有查詢、修改、刪除都同時帶 `user_id = ?`，`user_id` 一律取自 Token，不接受用戶端傳入。別人的任務一律回 404，不透露其是否存在。
- **輸入驗證**：註冊與任務使用獨立的輸入結構（只開放允許的欄位），避免用戶端自行指定 `role`、`user_id` 等欄位。
- **測試**：用記憶體資料庫測試註冊驗證、未授權存取、密碼不外洩、不能自選角色、使用者之間的資料隔離。
- **快取**：`GET /api/tasks` 使用 Cache-Aside，快取 60 秒；新增、修改、刪除任務時清除該使用者的快取。每位使用者獨立的 key，不會互相影響。Redis 不可用時自動退回直接查資料庫。
- **聊天室**：用單一 Hub goroutine 管理所有連線，以 channel 傳遞加入、離開與廣播，不需要加鎖；限制訊息大小與長度，並定期 ping 偵測斷線。

## 已知限制與規劃

- [x] 改用 PostgreSQL，並以 Docker Compose 啟動 app 與資料庫
- [x] GitHub Actions 自動建置與執行測試
- [x] 聊天室與 Redis 快取的自動化測試
- [ ] 聊天室訊息保存（例如存入資料庫或 Redis）
- [ ] 分頁、統一的錯誤回應格式、結構化 log
- [ ] Refresh Token 與登入失敗次數限制
- [ ] API 文件（Swagger 或 Postman collection）