package main

import (
	"go-backend-lab/database"
	"go-backend-lab/handlers"
	"go-backend-lab/middlewares"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. 初始化資料庫
	database.InitDB()

	// 2. 初始化 Gin 路由器
	r := gin.Default()

	// --- 公開路由 (不需要 Token 就能存取) ---
	r.POST("/api/register", handlers.Register)
	r.POST("/api/login", handlers.Login)
	r.POST("/api/heavy-task", handlers.HeavyTask)

	// --- 受保護的路由群組 (必須通過 AuthMiddleware 安檢門才能存取) ---
	protected := r.Group("/api")
	protected.Use(middlewares.AuthMiddleware())
	{
		protected.GET("/users", handlers.GetUsers) // 查詢清單現在需要 Token 了！

		protected.POST("/tasks", handlers.CreateTask)
		protected.GET("/tasks", handlers.GetTasks)
	}

	// 3. 啟動伺服器
	r.Run(":8080")
}
