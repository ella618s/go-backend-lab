package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email" gorm:"unique"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role"`
	Tasks    []Task // 一對多關聯：一個使用者有多個任務
}
