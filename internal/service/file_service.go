package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

type FileInfo struct {
	Name    string
	IsDir   bool
	ModTime time.Time
}

// FileService 定义存储接口
type FileService interface {
	// Put 保存文件到指定子路径
	Put(fileName string, data []byte, subPath string) (string, error)

	// Delete 删除指定子路径的文件
	Delete(subPath string) error

	// Get 获取指定子路径的文件绝对路径
	Get(subPath string) (string, error)

	// List 列出该目录下所有文件名
	List(subPath string) ([]FileInfo, error)
}

// LocalFileService 本地存储实现
type LocalFileService struct {
	URL      string
	Route    string
	BasePath string
}

func NewLocalFileService(app fiber.Router, url string, route string, basePath string) *LocalFileService {
	err := os.MkdirAll(basePath, os.ModePerm)
	if err != nil {
		return nil
	}
	app.Static(route, basePath)
	return &LocalFileService{URL: url, Route: route, BasePath: basePath}
}

// Put 保存文件
func (l *LocalFileService) Put(fileName string, data []byte, subPath string) (string, error) {
	fullPath := filepath.Join(l.BasePath, subPath, fileName)
	err := os.MkdirAll(filepath.Dir(fullPath), os.ModePerm)

	if err != nil {
		return "", err
	}

	if err := os.WriteFile(fullPath, data, os.ModePerm); err != nil {
		return "", err
	}

	// 需要返回 URL
	path := strings.ReplaceAll(filepath.Join(l.Route, subPath, fileName), "\\", "/")
	return fmt.Sprintf("%s%s", l.URL, path), nil
}

// Delete 删除文件
func (l *LocalFileService) Delete(subPath string) error {
	fullPath := filepath.Join(l.BasePath, subPath)
	if _, err := os.Stat(fullPath); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return os.Remove(fullPath)
}

// List 列出目录下所有文件及修改时间
func (l *LocalFileService) List(subPath string) ([]FileInfo, error) {
	fullPath := filepath.Join(l.BasePath, subPath)

	info, err := os.Stat(fullPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s 不是目录", fullPath)
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, err
	}

	var files []FileInfo
	for _, entry := range entries {
		fi, err := entry.Info()
		if err != nil {
			continue
		}

		files = append(files, FileInfo{
			Name:    entry.Name(),
			IsDir:   entry.IsDir(),
			ModTime: fi.ModTime(),
		})
	}

	return files, nil
}

// Get 获取文件绝对路径
func (l *LocalFileService) Get(subPath string) (string, error) {
	fullPath := filepath.Join(l.BasePath, subPath)
	if _, err := os.Stat(fullPath); errors.Is(err, os.ErrNotExist) {
		return "", errors.New("file not found")
	}

	// 需要返回 URL
	path := strings.ReplaceAll(filepath.Join(l.Route, subPath), "\\", "/")
	return fmt.Sprintf("%s%s", l.URL, path), nil
}
