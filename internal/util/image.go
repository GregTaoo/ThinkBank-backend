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
	"log"
	"path/filepath"
	"strings"
	"time"

	"archive/zip"

	"github.com/jdeng/goheif"
	"github.com/rwcarlsen/goexif/exif"
)

type ExifInfo struct {
	Latitude  float64
	Longitude float64
	CreateAt  time.Time
}

func ProcessImageToJPEG(data []byte, ext string) ([]byte, *ExifInfo, error) {
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

func encodeJPEGFromHEIC(data []byte) ([]byte, *ExifInfo, error) {
	exifInfo := extractHEICExifInfo(data)
	img, err := goheif.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, nil, err
	}
	var buf bytes.Buffer
	err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90})
	return buf.Bytes(), exifInfo, err
}

func encodeJPEGFromImageData(data []byte) ([]byte, *ExifInfo, error) {
	exifInfo := extractExifInfo(data)
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, nil, err
	}
	var buf bytes.Buffer
	err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90})
	return buf.Bytes(), exifInfo, err
}

// livp 内部递归处理图片或 heic
func extractImageFromLivpRecursive(data []byte) ([]byte, *ExifInfo, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, nil, err
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
			return nil, nil, err
		}

		return ProcessImageToJPEG(content, filepath.Ext(f.Name))
	}

	return nil, nil, fmt.Errorf("no image found in livp")
}

func extractExifInfo(data []byte) *ExifInfo {
	var err error
	x, err := exif.Decode(bytes.NewReader(data))
	if err != nil {
		log.Println("Error occurs while extracting EXIF:", err)
		return nil
	}

	tm, err := x.DateTime()
	lat, long, err := x.LatLong()
	if err != nil {
		log.Println("Error occurs while extracting EXIF DateTime and LatLong:", err)
		return nil
	}

	exifInfo := new(ExifInfo)
	exifInfo.Latitude = lat
	exifInfo.Longitude = long
	exifInfo.CreateAt = tm

	return exifInfo
}

func extractHEICExifInfo(data []byte) *ExifInfo {
	exifBytes, err := goheif.ExtractExif(bytes.NewReader(data))
	if err != nil {
		log.Println("Warning: no EXIF found", err)
	}
	return extractExifInfo(exifBytes)
}
