package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"go-backend-lab/cache"
	"go-backend-lab/database"
	"go-backend-lab/middlewares"
	"go-backend-lab/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// taskInput 只允許用戶端填寫這三個欄位，user_id 一律取自 Token。
type taskInput struct {
	Title       string `json:"title" binding:"required,max=200"`
	Description string `json:"description" binding:"max=2000"`
	Status      string `json:"status" binding:"omitempty,oneof=pending done"`
}

// currentUserID 從 Context 取出 AuthMiddleware 放進去的使用者 ID。
func currentUserID(c *gin.Context) (uint, bool) {
	v, exists := c.Get(middlewares.ContextUserID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "無法取得使用者身份"})
		return 0, false
	}
	id, ok := v.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "無法取得使用者身份"})
		return 0, false
	}
	return id, true
}

// CreateTask 新增任務，擁有者是目前登入的使用者。
func CreateTask(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var in taskInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task := models.Task{
		Title:       in.Title,
		Description: in.Description,
		Status:      in.Status,
		UserID:      userID,
	}
	if task.Status == "" {
		task.Status = "pending"
	}

	if err := database.DB.Create(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "建立任務失敗"})
		return
	}
	cache.InvalidateTasks(c.Request.Context(), userID)

	c.JSON(http.StatusCreated, gin.H{"message": "新增任務成功", "task": task})
}

// GetTasks 取得目前登入使用者自己的任務，可用 ?status=pending|done 篩選。
func GetTasks(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	status := c.Query("status")

	// 只快取合法的篩選值，避免有人用亂填的 status 在 Redis 塞出大量無用的資料
	field := "all"
	cacheable := status == ""
	if status == "pending" || status == "done" {
		field = status
		cacheable = true
	}

	// Cache-Aside：先看快取有沒有，有就直接回傳
	if cacheable {
		if data, hit := cache.GetTasks(c.Request.Context(), userID, field); hit {
			c.Header("X-Cache", "HIT")
			c.Data(http.StatusOK, "application/json; charset=utf-8", data)
			return
		}
	}

	query := database.DB.Where("user_id = ?", userID)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var tasks []models.Task
	if err := query.Order("id").Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查詢任務失敗"})
		return
	}

	body, err := json.Marshal(gin.H{"count": len(tasks), "tasks": tasks})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "回應格式化失敗"})
		return
	}
	if cacheable {
		cache.SetTasks(c.Request.Context(), userID, field, body)
	}

	c.Header("X-Cache", "MISS")
	c.Data(http.StatusOK, "application/json; charset=utf-8", body)
}

// UpdateTask 修改自己的任務。
func UpdateTask(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "任務 ID 格式錯誤"})
		return
	}

	var in taskInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 同時比對 id 與 user_id：別人的任務查不到，一律回 404，不透露它是否存在
	var task models.Task
	err = database.DB.Where("id = ? AND user_id = ?", id, userID).First(&task).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "找不到任務"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查詢任務失敗"})
		return
	}

	task.Title = in.Title
	task.Description = in.Description
	if in.Status != "" {
		task.Status = in.Status
	}
	if err := database.DB.Save(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新任務失敗"})
		return
	}
	cache.InvalidateTasks(c.Request.Context(), userID)

	c.JSON(http.StatusOK, gin.H{"message": "更新任務成功", "task": task})
}

// DeleteTask 刪除自己的任務。
func DeleteTask(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "任務 ID 格式錯誤"})
		return
	}

	result := database.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&models.Task{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "刪除任務失敗"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "找不到任務"})
		return
	}
	cache.InvalidateTasks(c.Request.Context(), userID)

	c.JSON(http.StatusOK, gin.H{"message": "刪除任務成功"})
}
