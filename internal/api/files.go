package api

import (
	"ThinkBank-backend/internal/db"
	"ThinkBank-backend/internal/model"
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// RegisterFileListRoute 注册分页文件列表接口 /files/list
func RegisterFileListRoute(app fiber.Router) {
	app.Get("/files/list", func(c *fiber.Ctx) error {
		// 解析 page 和 pageSize 参数，默认 page=1, pageSize=20
		page := 1
		pageSize := 20

		if val := c.Query("page"); val != "" {
			if p, err := strconv.Atoi(val); err == nil && p > 0 {
				page = p
			} else if err != nil {
				log.Printf("invalid page parameter: %v", err)
			}
		}

		if val := c.Query("pageSize"); val != "" {
			if ps, err := strconv.Atoi(val); err == nil && ps > 0 {
				pageSize = ps
			} else if err != nil {
				log.Printf("invalid pageSize parameter: %v", err)
			}
		}

		offset := (page - 1) * pageSize

		var files []model.File
		if err := db.Instance().Limit(pageSize).Offset(offset).Order("created_at DESC").Find(&files).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		// 返回 JSON
		result := make([]map[string]interface{}, len(files))
		for i, f := range files {
			result[i] = map[string]interface{}{
				"id":               f.ID,
				"fileName":         f.FileName,
				"originalFilePath": f.OriginalFilePath,
				"filePath":         f.FilePath,
				"type":             f.Type,
				"caption":          f.Caption,
				"tags":             f.Tags,
				"createdAt":        f.CreatedAt,
				"updatedAt":        f.UpdatedAt,
			}
		}

		return c.JSON(fiber.Map{
			"page":     page,
			"pageSize": pageSize,
			"count":    len(files),
			"files":    result,
		})
	})
}
