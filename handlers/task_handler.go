package handlers

import (
	"net/http"

	"go-backend-lab/database"
	"go-backend-lab/models"

	"github.com/gin-gonic/gin"
)

// CreateTask 新增任務（需要知道是哪位登入的使用者建立的）
func CreateTask(c *gin.Context) {
	var task models.Task
	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 從 Gin 的 Context 中取得我們在 AuthMiddleware 塞進去的 userID
	userID, exists := c.Get("userID")
	if exists {
		// 將 uint 形態的使用者 ID 指定給 Task 的 UserID
		task.UserID = userID.(uint)
	}

	// 存入資料庫
	if result := database.DB.Create(&task); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "建立任務失敗"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "新增任務成功！",
		"task":    task,
	})
}

// GetTasks 取得當前登入使用者的任務列表
func GetTasks(c *gin.Context) {
	// 1. 從 Context 拿取登入者的 userID
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "無法取得使用者身份"})
		return
	}

	var tasks []models.Task

	// 2. ✨ 關鍵：使用 .Where("user_id = ?", userID) 限制只查詢自己的任務
	if result := database.DB.Where("user_id = ?", userID).Preload("User").Find(&tasks); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查詢任務失敗"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": len(tasks),
		"tasks": tasks,
	})
}
