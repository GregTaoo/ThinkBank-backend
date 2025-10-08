package api

import (
	"ThinkBank-backend/internal/queue"
	"ThinkBank-backend/internal/service"
	"ThinkBank-backend/internal/util"
	"fmt"
	"mime/multipart"
	"time"

	"ThinkBank-backend/internal/db"
	"ThinkBank-backend/internal/model"

	"github.com/gofiber/fiber/v2"
)

// RegisterUploadRoutes 注册上传路由
func RegisterUploadRoutes(app fiber.Router, fileService service.FileService) {
	app.Post("/upload", UploadHandler(fileService))
}

// UploadHandler 支持多文件上传，按数据库 ID 重命名，自动填充 Type
func UploadHandler(fileService service.FileService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		form, err := c.MultipartForm()
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "No files uploaded"})
		}

		files := form.File["files"]
		if len(files) == 0 {
			return c.Status(400).JSON(fiber.Map{"error": "No files uploaded"})
		}

		results := make([]map[string]interface{}, 0, len(files))

		for _, file := range files {
			record, err := processSingleFile(file, fileService)
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"error": err.Error()})
			}
			results = append(results, map[string]interface{}{
				"id":               record.ID,
				"fileName":         record.FileName,
				"originalFilePath": record.OriginalFilePath,
				"type":             record.Type,
			})
		}

		return c.JSON(fiber.Map{
			"message":   "files uploaded successfully",
			"fileCount": len(results),
			"files":     results,
		})
	}
}

// processSingleFile 处理单个文件上传逻辑
func processSingleFile(file *multipart.FileHeader, fileService service.FileService) (*model.File, error) {
	// 打开文件
	f, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			fmt.Printf("failed to close file: %v", cerr)
		}
	}()

	data := make([]byte, file.Size)
	if _, err := f.Read(data); err != nil {
		return nil, err
	}

	// 先在数据库创建记录
	fileRecord := &model.File{
		FileName: file.Filename,
		Type:     util.GetFileType(file.Filename),
	}
	if err := db.Instance().Create(fileRecord).Error; err != nil {
		return nil, err
	}

	// 构造存储路径
	subPath := time.Now().Format("2006/01/02")
	storedFileName := fmt.Sprintf("%d%s", fileRecord.ID, util.GetFileExt(file.Filename))

	// 保存文件
	savedPath, err := fileService.Put(storedFileName, data, subPath)
	if err != nil {
		db.Instance().Delete(fileRecord)
		return nil, err
	}

	// 更新数据库路径
	fileRecord.OriginalFilePath = savedPath
	if err := db.Instance().Where("id = ?", fileRecord.ID).Updates(fileRecord).Error; err != nil {
		return nil, err
	}

	queue.ProduceNormalizeFile(fileRecord.ID, savedPath)

	return fileRecord, nil
}
