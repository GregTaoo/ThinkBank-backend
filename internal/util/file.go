package util

import (
	"path/filepath"
	"strings"
)

var documentExtSet = map[string]struct{}{
	".doc":  {},
	".docx": {},
	".md":   {},
	".txt":  {},
	".log":  {},
	".ppt":  {},
	".pptx": {},
	".xls":  {},
	".xlsx": {},
	".pdf":  {},
}

var imageExtSet = map[string]struct{}{
	".jpg":  {},
	".jpeg": {},
	".png":  {},
	".webp": {},
	".gif":  {},
	".heic": {},
	".livp": {},
	".apng": {},
}

func GetFileTypeByExt(ext string) string {
	ext = strings.ToLower(ext)
	_, isDoc := documentExtSet[ext]
	if isDoc {
		return "document"
	}
	_, isImg := imageExtSet[ext]
	if isImg {
		return "image"
	}
	return "unknown"
}

func GetFileExt(fileName string) string {
	return strings.ToLower(filepath.Ext(fileName))
}

func GetFileType(fileName string) string {
	return GetFileTypeByExt(GetFileExt(fileName))
}
