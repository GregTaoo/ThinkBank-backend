package search

import (
	"ThinkBank-backend/internal/db"
	"ThinkBank-backend/internal/model"
	"ThinkBank-backend/internal/service"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/pgvector/pgvector-go"
)

// 生成随机文件名（保留原文件后缀）
func randomFilename(originalName string) (string, error) {
	ext := ""
	for i := len(originalName) - 1; i >= 0; i-- {
		if originalName[i] == '.' {
			ext = originalName[i:]
			break
		}
	}
	b := make([]byte, 8)
	_, err := rand.Read(b)
	return hex.EncodeToString(b) + ext, err
}

// RegisterSearchByImage 注册 /image/search 路由
func RegisterSearchByImage(app fiber.Router, modelService service.ModelService, fileService *service.LocalFileService) {
	app.Post("/image/search", func(c *fiber.Ctx) error {
		fileHeader, err := c.FormFile("image")
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "image is required"})
		}

		dataBytes := make([]byte, fileHeader.Size)
		file, err := fileHeader.Open()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "cannot open uploaded file"})
		}
		defer func() {
			if err := file.Close(); err != nil {
				log.Printf("failed to close uploaded file: %v", err)
			}
		}()

		if _, err := file.Read(dataBytes); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "cannot read uploaded file"})
		}

		tmpFileName, err := randomFilename(fileHeader.Filename)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "error generating random filename"})
		}

		tmpFilePath, err := fileService.Put(tmpFileName, dataBytes, "")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "cannot save temp file"})
		}

		topK := 10
		if val := c.FormValue("topK"); val != "" {
			if _, err := fmt.Sscanf(val, "%d", &topK); err != nil || topK <= 0 {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid topK value"})
			}
		}

		files, err := ByImage(tmpFilePath, modelService, topK)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		result := make([]map[string]interface{}, len(files))
		for i, f := range files {
			result[i] = map[string]interface{}{
				"caption":          f.Caption,
				"filename":         f.FileName,
				"originalFilePath": f.OriginalFilePath,
				"filePath":         f.FilePath,
				"type":             f.Type,
			}
		}

		return c.JSON(fiber.Map{"count": len(result), "files": result})
	})
}

// ByImage 使用 embedding + HNSW 索引直接搜索
func ByImage(imagePath string, modelService service.ModelService, topK int) ([]model.File, error) {
	_, embedding, err := modelService.AnalyzeImage(imagePath)
	if err != nil {
		return nil, err
	}

	var files []model.File
	err = db.Instance().Raw(`
        SELECT *, vector <-> ? AS distance
        FROM files
        ORDER BY vector <-> ? 
        LIMIT ?
    `, pgvector.NewVector(embedding), pgvector.NewVector(embedding), topK).Scan(&files).Error
	if err != nil {
		return nil, err
	}

	return files, nil
}
