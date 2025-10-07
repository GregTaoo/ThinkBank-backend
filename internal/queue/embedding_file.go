package queue

import (
	"ThinkBank-backend/internal/db"
	"ThinkBank-backend/internal/model"
	"ThinkBank-backend/internal/service"
	"log"
	"runtime"

	"github.com/pgvector/pgvector-go"
)

// ProduceEmbeddingFile 推送消息到 embedding_file 队列
func ProduceEmbeddingFile(id uint, path string) {
	GlobalQueue.Produce("embedding_file", Payload{
		ID:   id,
		Path: path,
	})
}

// ConsumeEmbeddingFile 启动 n 个并发消费者处理 embedding_file
func ConsumeEmbeddingFile(modelService service.ModelService, n int) {
	GlobalQueue.Consume("embedding_file", func(msg Message) {
		runtime.NumGoroutine()
		payload, ok := msg.Data.(Payload)
		if !ok {
			log.Println("invalid payload")
			return
		}

		caption, embedding, err := modelService.AnalyzeImage(payload.Path)
		if err != nil {
			log.Println("analyze image error:", err)
			return
		}

		// 更新数据库
		err = db.Instance().Model(&model.File{}).Where("id = ?", payload.ID).Updates(map[string]interface{}{
			"caption": caption,
			"vector":  pgvector.NewVector(embedding),
		}).Error
		if err != nil {
			log.Println("update db error:", err)
			return
		}
	}, n)
}
