# 使用官方的 Go 輕量映像檔作為基礎環境
FROM golang:1.23-alpine

# 設定容器內的工作目錄
WORKDIR /app

# 先複製依賴管理檔案，利用 Docker 快取加速後續建構
COPY go.mod go.sum ./
RUN go mod download

# 複製專案底下的所有程式碼到容器中
COPY . .

# 將 Go 程式編譯為名為 main 的執行檔
RUN go build -o main .

# 開放容器的 8080 連接埠
EXPOSE 8080

# 當容器啟動時，直接執行編譯好的 main 程式
CMD ["./main"]