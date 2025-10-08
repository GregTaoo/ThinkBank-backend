package queue

import (
	"ThinkBank-backend/internal/db"
	"ThinkBank-backend/internal/model"
	"ThinkBank-backend/internal/service"
	"ThinkBank-backend/internal/util"
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"time"
)

func ProduceNormalizeFile(id uint, filePath string) {
	payload := Payload{
		ID:   id,
		Path: filePath,
	}
	GlobalQueue.Produce("normalize_file", payload)
}

func ConsumeNormalizeFile(concurrency int, fromFS, toFS service.FileService) {
	GlobalQueue.RegisterConsumer("normalize_file", func(msg Message) {
		handleNormalizeFile(msg, fromFS, toFS)
	}, concurrency)
}

func handleNormalizeFile(msg Message, fromFS, toFS service.FileService) {
	payload, ok := msg.Data.(Payload)
	if !ok {
		fmt.Println("Invalid normalize_file payload")
		return
	}

	id := payload.ID
	originalPath := payload.Path

	normalizedPath := processFile(fromFS, toFS, originalPath, id)

	err := db.Instance().Model(&model.File{}).
		Where("id = ?", id).
		Update("file_path", normalizedPath).Error
	if err != nil {
		fmt.Println("Update file error:", err)
	}

	ProduceEmbeddingFile(id, normalizedPath)
}

func processFile(fromFS, toFS service.FileService, path string, id uint) string {
	resp, err := http.Get(path)
	if err != nil {
		log.Println("Failed to normalize file because of http.Get error:", err)
		return path
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println("Failed to normalize file because of Body.Close() error:", err)
		}
	}(resp.Body)

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Failed to normalize file because of ReadAll error:", err)
		return path
	}

	ext := util.GetFileExt(path)
	var newData []byte
	var newExt string

	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".heic", ".livp", ".apng":
		newData, err = util.ProcessImageToJPEG(data, ext)
		if err != nil {
			log.Println("Image process error:", err)
			return path
		}
		newExt = ".jpg"

	default:
		// 其他文件直接保存原数据
		newData = data
		newExt = ext
	}

	// 构造存储路径
	newSubPath := time.Now().Format("2006/01/02")
	storedFileName := fmt.Sprintf("%d%s", id, newExt)

	// 写入 toFS
	toPath, err := toFS.Put(storedFileName, newData, newSubPath)
	if err != nil {
		log.Println("Put file to filesystem error:", err)
		return path
	}

	return toPath
}
