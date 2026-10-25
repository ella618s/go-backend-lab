package models

import "gorm.io/gorm"

// Task 屬於某一位使用者（UserID 外鍵）。
// UserID 一律由 Token 決定，不接受用戶端在請求內容裡指定。
type Task struct {
	gorm.Model
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status" gorm:"default:pending"` // pending / done
	UserID      uint   `json:"user_id" gorm:"index;not null"`
}
