package model

import (
	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
)

type File struct {
	gorm.Model
	FileName         string            `gorm:"type:text"`        // 原文件名
	OriginalFilePath string            `gorm:"type:text"`        // 文件存储路径
	FilePath         string            `gorm:"type:text"`        // 文件存储路径
	Metadata         map[string]string `gorm:"type:jsonb"`       // 文件元数据，如作者、定位等
	Type             string            `gorm:"type:text"`        // image / document
	Caption          string            `gorm:"type:text"`        // 模型生成描述
	Tags             []string          `gorm:"type:jsonb"`       // 关键词列表
	Vector           *pgvector.Vector  `gorm:"type:vector(512)"` // embedding 向量
	TSV              string            `gorm:"-"`                // 用于倒排索引
}
