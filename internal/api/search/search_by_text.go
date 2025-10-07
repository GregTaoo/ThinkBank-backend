package search

import (
	"ThinkBank-backend/internal/db"
	"ThinkBank-backend/internal/model"
	"ThinkBank-backend/internal/service"
	"sort"

	"github.com/gofiber/fiber/v2"
	"github.com/pgvector/pgvector-go"
)

// RegisterSearchByText 注册 /text/search 路由
func RegisterSearchByText(app fiber.Router, modelService service.ModelService) {
	app.Post("/text/search", func(c *fiber.Ctx) error {
		var req struct {
			Query string `json:"query"`
			TopK  int    `json:"topK"`
		}

		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid request body",
			})
		}

		if req.TopK == 0 {
			req.TopK = 10
		}

		files, err := ByText(req.Query, modelService, req.TopK, 0.5)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		result := make([]map[string]interface{}, len(files))
		for i, f := range files {
			result[i] = map[string]interface{}{
				"caption":          f.Caption,
				"filename":         f.FileName,
				"originalFilePath": f.OriginalFilePath,
				"filePath":         f.FilePath,
				"type":             f.Type,
			}
		}

		return c.JSON(fiber.Map{
			"count": len(result),
			"files": result,
		})
	})
}

func topKText(query string, topK int) (map[uint]float64, error) {
	var results []struct {
		ID    uint
		Score float64
	}
	err := db.Instance().Raw(`
        SELECT id, ts_rank(tsv, websearch_to_tsquery('english', ?)) AS score
        FROM files
        WHERE tsv @@ websearch_to_tsquery('english', ?)
        ORDER BY score DESC
        LIMIT ?
    `, query, query, topK).Scan(&results).Error
	if err != nil {
		return nil, err
	}

	scores := make(map[uint]float64)
	for _, r := range results {
		scores[r.ID] = r.Score
	}
	return scores, nil
}

// -------------------- 向量搜索 --------------------
func topKVector(embedding []float32, topK int) (map[uint]float64, error) {
	var results []struct {
		ID       uint
		Distance float64
	}
	err := db.Instance().Raw(`
        SELECT id, vector <-> ? AS distance
        FROM files
        ORDER BY vector <-> ?
        LIMIT ?
    `, pgvector.NewVector(embedding), pgvector.NewVector(embedding), topK).Scan(&results).Error
	if err != nil {
		return nil, err
	}

	scores := make(map[uint]float64)
	for _, r := range results {
		scores[r.ID] = 1 / (1 + r.Distance) // 距离转相似度
	}
	return scores, nil
}

func ByText(query string, modelService service.ModelService, topK int, alpha float64) ([]model.File, error) {
	// 1. 生成 embedding
	embedding, err := modelService.AnalyzeText(query)
	if err != nil {
		return nil, err
	}

	// 2. 文本搜索 topK
	textScores, err := topKText(query, topK)
	if err != nil {
		return nil, err
	}

	// 3. 向量搜索 topK
	vectorScores, err := topKVector(embedding, topK)
	if err != nil {
		return nil, err
	}

	// 4. 融合分数
	finalScores := make(map[uint]float64)
	for id, vScore := range vectorScores {
		tScore := textScores[id]
		finalScores[id] = alpha*tScore + (1-alpha)*vScore
	}
	for id, tScore := range textScores {
		if _, ok := finalScores[id]; !ok {
			finalScores[id] = alpha * tScore
		}
	}

	// 5. 排序取前 topK
	type sf struct {
		ID    uint
		Score float64
	}
	var scoredList []sf
	for id, score := range finalScores {
		scoredList = append(scoredList, sf{ID: id, Score: score})
	}
	sort.Slice(scoredList, func(i, j int) bool {
		return scoredList[i].Score > scoredList[j].Score
	})
	if len(scoredList) > topK {
		scoredList = scoredList[:topK]
	}

	// 6. 查询文件信息
	var files []model.File
	var ids []uint
	for _, s := range scoredList {
		ids = append(ids, s.ID)
	}
	err = db.Instance().Where("id IN ?", ids).Find(&files).Error
	if err != nil {
		return nil, err
	}

	// 保持顺序一致
	idToFile := make(map[uint]model.File)
	for _, f := range files {
		idToFile[f.ID] = f
	}
	ordered := make([]model.File, 0, len(ids))
	for _, id := range ids {
		if f, ok := idToFile[id]; ok {
			ordered = append(ordered, f)
		}
	}

	return ordered, nil
}
