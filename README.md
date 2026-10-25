# go-backend-lab

使用 Go（Gin、GORM）實作的 RESTful API 練習專案：使用者註冊與登入、JWT 認證、任務管理，並確保每位使用者只能存取自己的資料。

> 這是學習用專案，目前使用 SQLite。下一步規劃改用 PostgreSQL 與 Docker Compose。

## 技術

- Go、Gin（HTTP 路由與輸入驗證）
- GORM + SQLite
- JWT（`golang-jwt/jwt/v5`）、bcrypt 密碼雜湊
- `net/http/httptest` 的 API 測試
- Dockerfile（多階段建置）

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
| `DB_PATH` | 否 | SQLite 檔案路徑，預設 `app.db` |
| `PORT` | 否 | 監聽埠，預設 `8080` |

### 用 Docker 執行

```bash
docker build -t go-backend-lab .
docker run --rm -p 8080:8080 -e JWT_SECRET=$(openssl rand -hex 32) go-backend-lab
```

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

## 已知限制與規劃

- [ ] 改用 PostgreSQL，並以 Docker Compose 啟動 app 與資料庫
- [ ] 分頁、統一的錯誤回應格式、結構化 log
- [ ] GitHub Actions 自動執行測試
- [ ] Refresh Token 與登入失敗次數限制
- [ ] API 文件（Swagger 或 Postman collection）
