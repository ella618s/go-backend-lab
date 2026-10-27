package router

import (
	"go-backend-lab/chat"
	"go-backend-lab/handlers"
	"go-backend-lab/middlewares"

	"github.com/gin-gonic/gin"
)

// New 建立並回傳設定好所有路由的 Gin 引擎。
// 把路由抽出來，main 與測試就能共用同一份設定。
func New() *gin.Engine {
	r := gin.Default()

	// 聊天室：Hub 在背景 goroutine 裡處理所有連線的加入、離開與廣播
	hub := chat.NewHub()
	go hub.Run()

	r.GET("/chat", handlers.ChatPage)

	api := r.Group("/api")

	// 公開路由：不需要 Token
	api.POST("/register", handlers.Register)
	api.POST("/login", handlers.Login)

	// WebSocket 聊天室：Token 從 ?token= 帶入，仍然要通過 AuthMiddleware
	api.GET("/ws", middlewares.TokenFromQuery(), middlewares.AuthMiddleware(), handlers.ChatWS(hub))

	// 受保護路由：必須通過 AuthMiddleware
	protected := api.Group("")
	protected.Use(middlewares.AuthMiddleware())
	{
		protected.GET("/me", handlers.Me)

		protected.POST("/tasks", handlers.CreateTask)
		protected.GET("/tasks", handlers.GetTasks)
		protected.PUT("/tasks/:id", handlers.UpdateTask)
		protected.DELETE("/tasks/:id", handlers.DeleteTask)

		protected.POST("/heavy-task", handlers.HeavyTask)
	}

	return r
}
