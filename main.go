package main

import (
	"log"
	"os"

	"go-backend-lab/cache"
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

	// 2. 初始化資料庫：有設定 DATABASE_URL 就用 PostgreSQL，否則用本機 SQLite
	if url := os.Getenv("DATABASE_URL"); url != "" {
		if err := database.InitPostgres(url); err != nil {
			log.Fatal(err)
		}
	} else if err := database.Init(getenv("DB_PATH", "app.db")); err != nil {
		log.Fatal(err)
	}

	// 3. 初始化 Redis 快取（選用）：沒設定或連不上就不快取，API 照常運作
	if err := cache.Init(os.Getenv("REDIS_ADDR")); err != nil {
		log.Printf("Redis 無法使用，改為不使用快取: %v", err)
	}

	// 4. 啟動伺服器
	r := router.New()
	if err := r.Run(":" + getenv("PORT", "8080")); err != nil {
		log.Fatal(err)
	}
}
