package api

import (
	"github.com/gin-gonic/gin"
)

// SetupRouter 设置路由
func SetupRouter(handler *Handler) *gin.Engine {
	r := gin.Default()

	// 静态文件
	r.Static("/static", "./web/static")

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	// SEO相关
	r.GET("/robots.txt", handler.GetRobotsTxt)
	r.GET("/sitemap.xml", handler.GetSitemap)

	// API路由组
	api := r.Group("/api")
	{
		// 认证相关
		auth := api.Group("/auth")
		{
			auth.POST("/register", handler.Register)
			auth.POST("/login", handler.Login)
			auth.POST("/refresh", handler.RefreshToken)
			auth.GET("/me", handler.authService.AuthMiddleware(), handler.GetCurrentUser)
		}

		// 需要认证的API
		authenticated := api.Group("")
		authenticated.Use(handler.authService.AuthMiddleware())
		{
			// 分类相关（需要管理员权限）
			categories := authenticated.Group("/categories")
			categories.Use(handler.authService.RoleMiddleware("admin"))
			{
				categories.POST("", handler.CreateCategory)
				categories.PUT("/:id", handler.UpdateCategory)
				categories.DELETE("/:id", handler.DeleteCategory)
			}

			// 分类相关（公开访问）
			publicCategories := api.Group("/categories")
			{
				publicCategories.GET("", handler.GetCategories)
				publicCategories.GET("/tree", handler.GetCategoryTree)
			}

			// 关键词相关（需要管理员权限）
			keywords := authenticated.Group("/keywords")
			keywords.Use(handler.authService.RoleMiddleware("admin"))
			{
				keywords.POST("/fetch", handler.FetchKeywords)
				keywords.GET("/search", handler.SearchKeywords)
				keywords.POST("/assign", handler.AssignKeywordToCategory)
			}

			// 文章相关（需要编辑权限）
			articles := authenticated.Group("/articles")
			articles.Use(handler.authService.RoleMiddleware("admin", "editor"))
			{
				articles.POST("/generate", handler.GenerateArticle)
				articles.POST("/batch-generate", handler.BatchGenerateArticles)
				articles.PUT("/:id", handler.UpdateArticle)
				articles.PUT("/:id/publish", handler.PublishArticle)
				articles.PUT("/:id/archive", handler.ArchiveArticle)
				articles.DELETE("/:id", handler.DeleteArticle)
			}

			// 任务相关（需要认证）
			tasks := authenticated.Group("/tasks")
			{
				tasks.GET("", handler.GetTaskList)
				tasks.GET("/:id", handler.GetTaskStatus)
			}

			// 文章相关（公开访问）
			publicArticles := api.Group("/articles")
			{
				publicArticles.GET("", handler.GetArticles)
				publicArticles.GET("/:id", handler.GetArticle)
				publicArticles.GET("/slug/:slug", handler.GetArticleBySlug)
			}
		}
	}

	// 前端页面路由
	r.GET("/health/:slug", func(c *gin.Context) {
		slug := c.Param("slug")
		c.HTML(200, "article.html", gin.H{
			"slug": slug,
		})
	})

	// 首页
	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", gin.H{})
	})

	return r
}
