package main

import (
	"ThinkBank-backend/internal/api"
	"ThinkBank-backend/internal/api/search"
	"ThinkBank-backend/internal/db"
	"ThinkBank-backend/internal/db/migrate"
	"ThinkBank-backend/internal/queue"
	"ThinkBank-backend/internal/service"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
)

func main() {
	if os.Getenv("GO_ENV") != "production" {
		// 测试环境下 .env
		if err := godotenv.Load(); err != nil {
			log.Fatal("Failed to load .env file")
		}

		if os.Getenv("USE_CGO") == "1" {
			// 启动 Profiling 工具 CGO
			go func() {
				log.Println(http.ListenAndServe("localhost:6060", nil))
			}()
		}
	}

	// PostgreSQL
	db.InitPostgres(
		os.Getenv("POSTGRE_USER"),
		os.Getenv("POSTGRE_PASSWORD"),
		os.Getenv("POSTGRE_DB"),
		os.Getenv("POSTGRE_HOST"),
		os.Getenv("POSTGRE_PORT"),
	)

	// 数据库的迁移
	migrate.InitExtensions()
	migrate.DBMigrateAll()
	migrate.InitIndices()

	// fiber 实例
	app := fiber.New(fiber.Config{
		BodyLimit: 100 * 1024 * 1024, // 100 MB
	})

	// CORS 中间件
	app.Use(cors.New(cors.Config{
		AllowOrigins: os.Getenv("FRONTEND_URL"),
		AllowMethods: "*",
		AllowHeaders: "*",
	}))

	backendURL := os.Getenv("BACKEND_URL")

	// 文件服务
	originalFileService := service.NewLocalFileService(
		app, backendURL, "/uploads/original", "./uploads/original")
	normalizedFileService := service.NewLocalFileService(
		app, backendURL, "/uploads/normalized", "./uploads/normalized")
	tmpFileService := service.NewLocalFileService(
		app, backendURL, "/uploads/tmp", "./uploads/tmp")

	service.RegisterFileCleaner(tmpFileService, "", 180*time.Second, 180*time.Second)

	// 模型服务
	modelService := service.NewHTTPModelService(os.Getenv("MODEL_SERVICE_URL"))

	// 路由
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, ThinkBank!")
	})

	api.RegisterUploadRoutes(app, originalFileService)
	api.RegisterFileListRoute(app)
	search.RegisterSearchByText(app, modelService)
	search.RegisterSearchByImage(app, modelService, tmpFileService)

	// 消息队列
	queue.ConsumeNormalizeFile(3, originalFileService, normalizedFileService)
	queue.ConsumeEmbeddingFile(modelService, 3)

	// 端口监听
	log.Fatal(app.Listen(fmt.Sprintf(":%s", os.Getenv("BACKEND_PORT"))))
}
