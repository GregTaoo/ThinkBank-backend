package util

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"path/filepath"
	"strings"

	"archive/zip"

	"github.com/jdeng/goheif"
)

func ProcessImageToJPEG(data []byte, ext string) ([]byte, error) {
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

// livp 内部递归处理图片或 heic
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
			strings.HasSuffix(lcName, ".heic")) {
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

		return ProcessImageToJPEG(content, filepath.Ext(f.Name))
	}

	return nil, fmt.Errorf("no image found in livp")
}
