package service

import (
	"log"
	"time"
)

func ClearFiles(fs FileService, subPath string, olderThan time.Duration) error {
	files, err := fs.List(subPath)
	if err != nil {
		return err
	}

	cutoff := time.Now().Add(-olderThan)

	for _, f := range files {
		if f.ModTime.Before(cutoff) {
			if err := fs.Delete(f.Name); err != nil {
				log.Println("文件删除失败:", f.Name, err)
			}
		}
	}

	return nil
}

func RegisterFileCleaner(fs FileService, subPath string, olderThan, interval time.Duration) {
	RegisterPeriodicService(func() {
		err := ClearFiles(fs, subPath, olderThan)
		if err != nil {
			log.Printf("Failed to clear files for %s: %s", subPath, err)
		}
	}, interval)
}
