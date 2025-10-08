package model

import (
	"time"

	"github.com/restayway/gogis"
)

type Geo struct {
	ID        uint        `gorm:"primaryKey;autoIncrement:false;uniqueIndex"`
	Latitude  float64     `gorm:"not null"`
	Longitude float64     `gorm:"not null"`
	Geom      gogis.Point `gorm:"type:geometry(Point,4326);index:idx_geo_geom_gist,type:gist"`
	CreateAt  time.Time
}
