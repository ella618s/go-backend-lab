package handlers

import (
	"log"
	"net/http"
	"time"

	"go-backend-lab/database"
	"go-backend-lab/models"

	"github.com/gin-gonic/gin"
)

// Me 回傳目前登入者自己的資料與任務清單（使用 Preload 一次載入一對多關聯）。
// 密碼雜湊因為 json:"-" 不會被輸出。
func Me(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var user models.User
	if err := database.DB.Preload("Tasks").First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "找不到使用者"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// 同時最多 5 個背景任務，避免有人狂送請求讓 goroutine 無限增加。
var heavyTaskSlots = make(chan struct{}, 5)

// HeavyTask 示範用 goroutine 在背景執行工作（需要登入）。
func HeavyTask(c *gin.Context) {
	var body struct {
		TaskName string `json:"task_name" binding:"required,max=100"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	select {
	case heavyTaskSlots <- struct{}{}:
	default:
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "背景任務已滿，請稍後再試"})
		return
	}

	go func(name string) {
		defer func() { <-heavyTaskSlots }()
		log.Printf("[背景任務] 開始: %s", name)
		time.Sleep(3 * time.Second)
		log.Printf("[背景任務] 完成: %s", name)
	}(body.TaskName)

	c.JSON(http.StatusAccepted, gin.H{
		"status":  "accepted",
		"message": "任務已接收，正在背景執行",
		"task":    body.TaskName,
	})
}
