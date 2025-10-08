package migrate

import (
	"ThinkBank-backend/internal/db"
	"fmt"
	"log"
)

const maxDegree = 16
const efConstruction = 200

func InitIndices() {
	sql := fmt.Sprintf(`
-- GIN 索引
ALTER TABLE files ADD COLUMN IF NOT EXISTS tsv tsvector;
CREATE INDEX IF NOT EXISTS idx_files_tsv
ON files USING gin(tsv);

-- HNSW 索引
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_indexes
        WHERE schemaname = 'public'
          AND tablename = 'files'
          AND indexname = 'idx_files_vector_hnsw'
    ) THEN
        CREATE INDEX idx_files_vector_hnsw
        ON files USING hnsw (vector vector_l2_ops)
        WITH (m = %d, ef_construction = %d);
    END IF;
END$$;
    `, maxDegree, efConstruction)

	if err := db.Instance().Exec(sql).Error; err != nil {
		log.Fatal("GIN / HNSW index initialization failed:", err)
	}
	log.Println("GIN / HNSW index initialized")
}
