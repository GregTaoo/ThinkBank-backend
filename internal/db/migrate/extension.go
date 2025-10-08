package migrate

import (
	"ThinkBank-backend/internal/db"
	"fmt"
	"log"
)

func InitExtensions() {
	sql := fmt.Sprintf(`
-- 扩展启用
CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS postgis;
    `)

	if err := db.Instance().Exec(sql).Error; err != nil {
		log.Fatal("Extensions initialization failed:", err)
	}
	log.Println("Extensions index initialized")
}
