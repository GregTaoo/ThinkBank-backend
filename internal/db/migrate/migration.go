package migrate

import (
	"ThinkBank-backend/internal/db"
	"ThinkBank-backend/internal/model"
	"log"
)

// DBMigrateAll 用于迁移所有表结构
func DBMigrateAll() {
	log.Println("Starting table migrations")

	if err := db.Instance().AutoMigrate(&model.File{}); err != nil {
		log.Fatal("Files table migration failed:", err)
	}

	log.Println("Table migrations completed")
}
