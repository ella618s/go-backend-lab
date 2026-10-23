package models

import "gorm.io/gorm"

type Task struct {
	gorm.Model
	Title       string `json:"title" binding:"required"` // 任務標題（必填）
	Description string `json:"description"`              // 任務描述
	Status      string `json:"status"`                   // 狀態 (例如: pending / done)
	UserID      uint   `json:"user_id"`                  // 外鍵：對應到使用者的 ID
	User        User   `json:"user"`
}
