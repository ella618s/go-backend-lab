package router

import (
	"go-backend-lab/handlers"
	"go-backend-lab/middlewares"

	"github.com/gin-gonic/gin"
)

// New 建立並回傳設定好所有路由的 Gin 引擎。
// 把路由抽出來，main 與測試就能共用同一份設定。
func New() *gin.Engine {
	r := gin.Default()

	api := r.Group("/api")

	// 公開路由：不需要 Token
	api.POST("/register", handlers.Register)
	api.POST("/login", handlers.Login)

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
