package main

import (
	"log"
	"os"

	"go-backend-lab/database"
	"go-backend-lab/middlewares"
	"go-backend-lab/router"
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	// 1. 設定 JWT 密鑰（必須由環境變數提供）
	if err := middlewares.SetJWTSecret(os.Getenv("JWT_SECRET")); err != nil {
		log.Fatal(err)
	}

	// 2. 初始化資料庫
	if err := database.Init(getenv("DB_PATH", "app.db")); err != nil {
		log.Fatal(err)
	}

	// 3. 啟動伺服器
	r := router.New()
	if err := r.Run(":" + getenv("PORT", "8080")); err != nil {
		log.Fatal(err)
	}
}
