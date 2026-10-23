package handlers

import (
	"log"
	"net/http"
	"time"

	"go-backend-lab/database"
	"go-backend-lab/models"

	"github.com/gin-gonic/gin"
)

// CreateUser 新增使用者
func CreateUser(c *gin.Context) {
	var user models.User

	// 如果前端傳的資料不符合 binding 規則，這裡的 err 就會有內容
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "資料驗證失敗，請檢查欄位格式！",
			"details": err.Error(),
		})
		return
	}

	database.DB.Create(&user)
	c.JSON(http.StatusOK, gin.H{"message": "使用者建立成功！", "data": user})
}

// GetUsers 取得所有使用者列表
func GetUsers(c *gin.Context) {
	var users []models.User
	database.DB.Find(&users)
	c.JSON(http.StatusOK, gin.H{
		"count": len(users),
		"data":  users,
	})
}

// HeavyTask 模擬背景非同步任務 (Goroutine)
func HeavyTask(c *gin.Context) {
	var body struct {
		TaskName string `json:"task_name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 啟動背景協程
	go func(task string) {
		log.Printf(">>> [背景協程] 開始執行任務: %s", task)
		time.Sleep(3 * time.Second)
		log.Printf(">>> [背景協程] 任務完成: %s", task)
	}(body.TaskName)

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "任務已成功接收，正在背景執行中！",
		"task":    body.TaskName,
	})
}
