# --- 建置階段 ---
# go-sqlite3 需要 cgo，所以要安裝 C 編譯器（build-base）。
# 沒有 cgo 的話程式雖然編得過，但一執行就會因為 SQLite 無法使用而失敗。
# Go 版本請與 go.mod 的 go 行一致。
FROM golang:1.27-alpine AS build
RUN apk add --no-cache build-base
WORKDIR /app

# 先複製依賴檔，利用 Docker 快取加速後續建構
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -o /out/server .

# --- 執行階段（只放編譯好的執行檔，映像檔較小）---
FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/server .
EXPOSE 8080

# JWT_SECRET 不寫在映像檔裡，啟動時用 -e 或 --env-file 傳入
CMD ["./server"]
