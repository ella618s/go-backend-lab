package database

import (
	"fmt"

	"go-backend-lab/models"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

// Init 開啟 SQLite 並自動建立資料表。dsn 是 SQLite 的檔案路徑（測試用 ":memory:"）。
// 失敗時回傳錯誤，由呼叫端決定如何處理，而不是直接 panic。
func Init(dsn string) error {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("開啟資料庫失敗: %w", err)
	}
	return migrate(db)
}

// InitPostgres 連線到 PostgreSQL 並自動建立資料表。
// dsn 例如：postgres://user:password@host:5432/dbname?sslmode=disable
func InitPostgres(dsn string) error {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("連線 PostgreSQL 失敗: %w", err)
	}
	return migrate(db)
}

func migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&models.User{}, &models.Task{}); err != nil {
		return fmt.Errorf("資料表遷移失敗: %w", err)
	}
	DB = db
	return nil
}
