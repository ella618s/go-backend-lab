package models

import "gorm.io/gorm"

// User 是資料庫中的使用者資料表。
// Password 存的是 bcrypt 雜湊，且 json:"-" 確保它永遠不會出現在任何 API 回應裡。
type User struct {
	gorm.Model
	Name     string `json:"name"`
	Email    string `json:"email" gorm:"unique;not null"`
	Password string `json:"-" gorm:"not null"`
	Role     string `json:"role" gorm:"default:user"`
	Tasks    []Task `json:"tasks,omitempty"` // 一對多：一個使用者有多個任務
}
