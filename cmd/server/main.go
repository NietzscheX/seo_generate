package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/NietzscheX/seo-generate/config"
	"github.com/NietzscheX/seo-generate/internal/api"
	"github.com/NietzscheX/seo-generate/internal/database"
	"github.com/NietzscheX/seo-generate/internal/models"
	"github.com/NietzscheX/seo-generate/internal/services"
	"github.com/NietzscheX/seo-generate/pkg/seo"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 设置Gin模式
	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 连接数据库
	db, err := database.NewDatabase(cfg)
	if err != nil {
		log.Fatalf("xxx连接数据库失败: %v", err)
	}

	// 连接Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// 测试Redis连接
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("连接Redis失败: %v", err)
	}

	// 自动迁移数据库表结构
	if err := models.AutoMigrate(db); err != nil {
		log.Fatalf("迁移数据库表结构失败: %v", err)
	}

	// 初始化服务
	categoryService := services.NewCategoryService(db)
	keywordService := services.NewKeywordService(db, cfg)
	contentService := services.NewContentService(db, cfg)
	articleService := services.NewArticleService(db)
	seoService := seo.NewSEOService(cfg)
	authService := services.NewAuthService(db, cfg)
	queueService := services.NewQueueService(db, rdb, cfg, contentService)

	// 初始化默认分类
	if err := categoryService.InitDefaultCategories(); err != nil {
		log.Printf("初始化默认分类失败: %v", err)
	}

	// 创建默认管理员用户
	adminUser := services.RegisterRequest{
		Username: "admin",
		Email:    "admin@example.com",
		Password: "admin123",
	}
	if _, err := authService.Register(adminUser); err != nil {
		if !strings.Contains(err.Error(), "用户名已存在") {
			log.Printf("创建管理员用户失败: %v", err)
		}
	} else {
		// 设置为管理员角色
		var user models.User
		if err := db.Where("username = ?", adminUser.Username).First(&user).Error; err == nil {
			user.Role = "admin"
			db.Save(&user)
		}
	}

	// 初始化API处理器
	handler := api.NewHandler(
		cfg,
		keywordService,
		categoryService,
		contentService,
		articleService,
		seoService,
		authService,
		queueService,
	)

	// 设置路由
	router := api.SetupRouter(handler)

	// 加载HTML模板
	router.LoadHTMLGlob("web/templates/*")

	// 启动任务处理器
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go queueService.ProcessTasks(ctx)

	// 创建HTTP服务器
	server := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: router,
	}

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 启动服务器
	go func() {
		log.Printf("服务器启动在 http://localhost:%s", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("启动服务器失败: %v", err)
		}
	}()

	// 等待中断信号
	<-quit
	log.Println("正在关闭服务器...")

	// 取消任务处理器
	cancel()

	// 设置关闭超时
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 关闭服务器
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("服务器关闭失败: %v", err)
	}

	log.Println("服务器已关闭")
}
