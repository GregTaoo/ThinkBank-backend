package queue

import (
	"ThinkBank-backend/internal/db"
	"ThinkBank-backend/internal/model"
	"ThinkBank-backend/internal/service"
	"archive/zip"
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/jdeng/goheif"
)

func ProduceNormalizeFile(id uint, filePath string) {
	payload := Payload{
		ID:   id,
		Path: filePath,
	}
	GlobalQueue.Produce("normalize_file", payload)
}

func ConsumeNormalizeFile(concurrency int, fromFS, toFS service.FileService) {
	GlobalQueue.Consume("normalize_file", func(msg Message) {
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
		log.Println("http.Get error:", err)
		return path
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println("Body.Close() error:", err)
		}
	}(resp.Body)

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("ReadAll error:", err)
		return path
	}

	ext := strings.ToLower(filepath.Ext(path))
	var newData []byte
	var newExt string

	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".heic", ".livp":
		newData, err = processImageBytes(data, ext)
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
		fmt.Println("Put file to toFS error:", err)
		return path
	}

	return toPath
}

// ---------------- 图片处理 ----------------
func processImageBytes(data []byte, ext string) ([]byte, error) {
	ext = strings.ToLower(ext)

	switch ext {
	case ".heic":
		return encodeJPEGFromHEIC(data)
	case ".livp":
		return extractImageFromLivpRecursive(data)
	default: // jpg/png/gif
		return encodeJPEGFromImageData(data)
	}
}

func encodeJPEGFromHEIC(data []byte) ([]byte, error) {
	img, err := goheif.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90})
	return buf.Bytes(), err
}

func encodeJPEGFromImageData(data []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90})
	return buf.Bytes(), err
}

// 支持 livp 内部递归处理图片或 heic
func extractImageFromLivpRecursive(data []byte) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}

	for _, f := range r.File {
		lcName := strings.ToLower(f.Name)
		if !(strings.HasSuffix(lcName, ".png") ||
			strings.HasSuffix(lcName, ".jpg") ||
			strings.HasSuffix(lcName, ".jpeg") ||
			strings.HasSuffix(lcName, ".gif") ||
			strings.HasSuffix(lcName, ".heic") ||
			strings.HasSuffix(lcName, ".livp")) {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			continue
		}
		content, err := io.ReadAll(rc)
		if err != nil {
			continue
		}

		err = rc.Close()
		if err != nil {
			return nil, err
		}

		return processImageBytes(content, filepath.Ext(f.Name))
	}

	return nil, fmt.Errorf("no image found in livp")
}
