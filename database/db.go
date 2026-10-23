package database

import (
	"go-backend-lab/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	var err error
	DB, err = gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		panic("無法連線到資料庫！")
	}

	// 自動遷移建立資料表
	DB.AutoMigrate(&models.User{}, &models.Task{})
}
