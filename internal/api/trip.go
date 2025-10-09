package api

import (
	"ThinkBank-backend/internal/db"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/lib/pq"
)

type TripCluster struct {
	Level      string        `json:"level"`
	Period     time.Time     `json:"period"`
	CenterLon  float64       `json:"center_lon"`
	CenterLat  float64       `json:"center_lat"`
	PhotoCount int           `json:"photo_count"`
	StartTs    time.Time     `json:"start_ts"`
	EndTs      time.Time     `json:"end_ts"`
	PhotoIDs   pq.Int64Array `json:"photo_ids" gorm:"type:integer[]"`
}

// RegisterTripRoutes 注册上传路由
func RegisterTripRoutes(app fiber.Router) {
	app.Get("/trip", func(c *fiber.Ctx) error {
		clusters, err := QueryTrips()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(clusters)
	})
}

func QueryTrips() ([]TripCluster, error) {
	sql := `
WITH params AS (
  SELECT
    20000.0::double precision AS eps_day,
    150000.0::double precision AS eps_week,
    800000.0::double precision AS eps_month
),

clusters_day AS (
  SELECT 
    date_trunc('day', create_at)::date AS period,
    unnest(ST_ClusterWithin(geom3857, params.eps_day)) AS cluster_geom_3857
  FROM geos, params
  GROUP BY date_trunc('day', create_at)
),

clusters_week AS (
  SELECT 
    date_trunc('week', create_at)::date AS period,
    unnest(ST_ClusterWithin(geom3857, params.eps_week)) AS cluster_geom_3857
  FROM geos, params
  GROUP BY date_trunc('week', create_at)
),

clusters_month AS (
  SELECT 
    date_trunc('month', create_at)::date AS period,
    unnest(ST_ClusterWithin(geom3857, params.eps_month)) AS cluster_geom_3857
  FROM geos, params
  GROUP BY date_trunc('month', create_at)
),

all_clusters AS (
  SELECT 'day' AS level, period, cluster_geom_3857 FROM clusters_day
  UNION ALL
  SELECT 'week' AS level, period, cluster_geom_3857 FROM clusters_week
  UNION ALL
  SELECT 'month' AS level, period, cluster_geom_3857 FROM clusters_month
)

SELECT
  level,
  period,
  ST_X(ST_Transform(ST_Centroid(cluster_geom_3857), 4326)) AS center_lon,
  ST_Y(ST_Transform(ST_Centroid(cluster_geom_3857), 4326)) AS center_lat,
  COUNT(g.id) AS photo_count,
  MIN(g.create_at) AS start_ts,
  MAX(g.create_at) AS end_ts,
  COALESCE(array_agg(g.id), '{}') AS photo_ids
FROM all_clusters c
JOIN geos g 
  ON ST_Intersects(g.geom3857, c.cluster_geom_3857)
GROUP BY level, period, c.cluster_geom_3857
HAVING COUNT(g.id) >= 5;
`
	var clusters []TripCluster
	if err := db.Instance().Raw(sql).Scan(&clusters).Error; err != nil {
		return nil, err
	}
	return clusters, nil
}
