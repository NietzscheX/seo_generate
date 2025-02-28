package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/NietzscheX/seo-generate/config"
	"github.com/NietzscheX/seo-generate/internal/models"
	"github.com/NietzscheX/seo-generate/internal/services"
	"github.com/NietzscheX/seo-generate/pkg/seo"
	"github.com/gin-gonic/gin"
)

// Handler API处理器
type Handler struct {
	config          *config.Config
	keywordService  *services.KeywordService
	categoryService *services.CategoryService
	contentService  *services.ContentService
	articleService  *services.ArticleService
	seoService      *seo.SEOService
	authService     *services.AuthService
	queueService    *services.QueueService
}

// NewHandler 创建API处理器
func NewHandler(
	cfg *config.Config,
	keywordService *services.KeywordService,
	categoryService *services.CategoryService,
	contentService *services.ContentService,
	articleService *services.ArticleService,
	seoService *seo.SEOService,
	authService *services.AuthService,
	queueService *services.QueueService,
) *Handler {
	return &Handler{
		config:          cfg,
		keywordService:  keywordService,
		categoryService: categoryService,
		contentService:  contentService,
		articleService:  articleService,
		seoService:      seoService,
		authService:     authService,
		queueService:    queueService,
	}
}

// Response 通用响应
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    data,
	})
}

// Error 错误响应
func Error(c *gin.Context, code int, message string) {
	c.JSON(code, Response{
		Code:    code,
		Message: message,
	})
}

// PaginationResponse 分页响应
type PaginationResponse struct {
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
	Items    interface{} `json:"items"`
}

// GetCategories 获取分类列表
func (h *Handler) GetCategories(c *gin.Context) {
	categories, err := h.categoryService.GetAllCategories()
	if err != nil {
		Error(c, http.StatusInternalServerError, "获取分类失败: "+err.Error())
		return
	}

	Success(c, categories)
}

// GetCategoryTree 获取分类树
func (h *Handler) GetCategoryTree(c *gin.Context) {
	categories, err := h.categoryService.GetCategoryTree()
	if err != nil {
		Error(c, http.StatusInternalServerError, "获取分类树失败: "+err.Error())
		return
	}

	Success(c, categories)
}

// CreateCategory 创建分类
func (h *Handler) CreateCategory(c *gin.Context) {
	var req struct {
		Name     string `json:"name" binding:"required"`
		ParentID *uint  `json:"parent_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "无效的请求参数: "+err.Error())
		return
	}

	category, err := h.categoryService.CreateCategory(req.Name, req.ParentID)
	if err != nil {
		Error(c, http.StatusInternalServerError, "创建分类失败: "+err.Error())
		return
	}

	Success(c, category)
}

// UpdateCategory 更新分类
func (h *Handler) UpdateCategory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		Error(c, http.StatusBadRequest, "无效的分类ID")
		return
	}

	var req struct {
		Name     string `json:"name" binding:"required"`
		ParentID *uint  `json:"parent_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "无效的请求参数: "+err.Error())
		return
	}

	category, err := h.categoryService.UpdateCategory(uint(id), req.Name, req.ParentID)
	if err != nil {
		Error(c, http.StatusInternalServerError, "更新分类失败: "+err.Error())
		return
	}

	Success(c, category)
}

// DeleteCategory 删除分类
func (h *Handler) DeleteCategory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		Error(c, http.StatusBadRequest, "无效的分类ID")
		return
	}

	if err := h.categoryService.DeleteCategory(uint(id)); err != nil {
		Error(c, http.StatusInternalServerError, "删除分类失败: "+err.Error())
		return
	}

	Success(c, nil)
}

// FetchKeywords 获取关键词
func (h *Handler) FetchKeywords(c *gin.Context) {
	var req struct {
		Category string `json:"category" binding:"required"`
		Limit    int    `json:"limit"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "无效的请求参数: "+err.Error())
		return
	}

	if req.Limit <= 0 {
		req.Limit = 100
	}

	keywords, err := h.keywordService.FetchKeywordsByCategory(req.Category, req.Limit)
	if err != nil {
		Error(c, http.StatusInternalServerError, "获取关键词失败: "+err.Error())
		return
	}

	Success(c, keywords)
}

// SearchKeywords 搜索关键词
func (h *Handler) SearchKeywords(c *gin.Context) {
	query := c.Query("q")
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")

	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	keywords, total, err := h.keywordService.SearchKeywords(query, page, pageSize)
	if err != nil {
		Error(c, http.StatusInternalServerError, "搜索关键词失败: "+err.Error())
		return
	}

	Success(c, PaginationResponse{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Items:    keywords,
	})
}

// AssignKeywordToCategory 将关键词分配到分类
func (h *Handler) AssignKeywordToCategory(c *gin.Context) {
	var req struct {
		KeywordID  uint `json:"keyword_id" binding:"required"`
		CategoryID uint `json:"category_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "无效的请求参数: "+err.Error())
		return
	}

	if err := h.keywordService.AssignKeywordToCategory(req.KeywordID, req.CategoryID); err != nil {
		Error(c, http.StatusInternalServerError, "分配关键词失败: "+err.Error())
		return
	}

	Success(c, nil)
}

// GenerateArticle 生成文章
func (h *Handler) GenerateArticle(c *gin.Context) {
	var req struct {
		KeywordID   uint   `json:"keyword_id" binding:"required"`
		CategoryIDs []uint `json:"category_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "无效的请求参数: "+err.Error())
		return
	}

	// 获取关键词
	keyword, err := h.keywordService.GetKeywordByID(req.KeywordID)
	if err != nil {
		Error(c, http.StatusInternalServerError, "获取关键词失败: "+err.Error())
		return
	}

	// 创建上下文，设置超时
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(h.config.AI.Timeout)*time.Second)
	defer cancel()

	// 生成文章
	article, err := h.contentService.GenerateArticle(ctx, *keyword, req.CategoryIDs)
	if err != nil {
		Error(c, http.StatusInternalServerError, "生成文章失败: "+err.Error())
		return
	}

	Success(c, article)
}

// GetArticles 获取文章列表
func (h *Handler) GetArticles(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")
	categoryIDStr := c.Query("category_id")
	status := c.DefaultQuery("status", "published")

	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	var categoryID *uint
	if categoryIDStr != "" {
		id, err := strconv.ParseUint(categoryIDStr, 10, 32)
		if err == nil {
			uintID := uint(id)
			categoryID = &uintID
		}
	}

	articles, total, err := h.articleService.GetArticles(page, pageSize, categoryID, status)
	if err != nil {
		Error(c, http.StatusInternalServerError, "获取文章失败: "+err.Error())
		return
	}

	Success(c, PaginationResponse{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Items:    articles,
	})
}

// GetArticle 获取文章详情
func (h *Handler) GetArticle(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		Error(c, http.StatusBadRequest, "无效的文章ID")
		return
	}

	article, err := h.articleService.GetArticleByID(uint(id))
	if err != nil {
		Error(c, http.StatusInternalServerError, "获取文章失败: "+err.Error())
		return
	}

	Success(c, article)
}

// GetArticleBySlug 根据Slug获取文章
func (h *Handler) GetArticleBySlug(c *gin.Context) {
	slug := c.Param("slug")

	article, err := h.articleService.GetArticleBySlug(slug)
	if err != nil {
		Error(c, http.StatusInternalServerError, "获取文章失败: "+err.Error())
		return
	}

	// 生成结构化数据
	schema := h.seoService.GenerateArticleSchema(article)
	schemaJSON, _ := json.Marshal(schema)

	// 返回文章和结构化数据
	Success(c, gin.H{
		"article": article,
		"schema":  string(schemaJSON),
	})
}

// UpdateArticle 更新文章
func (h *Handler) UpdateArticle(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		Error(c, http.StatusBadRequest, "无效的文章ID")
		return
	}

	var req struct {
		Title       string `json:"title" binding:"required"`
		Content     string `json:"content" binding:"required"`
		Summary     string `json:"summary"`
		MetaTitle   string `json:"meta_title"`
		MetaDesc    string `json:"meta_desc"`
		CategoryIDs []uint `json:"category_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "无效的请求参数: "+err.Error())
		return
	}

	article, err := h.articleService.UpdateArticle(
		uint(id),
		req.Title,
		req.Content,
		req.Summary,
		req.MetaTitle,
		req.MetaDesc,
		req.CategoryIDs,
	)
	if err != nil {
		Error(c, http.StatusInternalServerError, "更新文章失败: "+err.Error())
		return
	}

	Success(c, article)
}

// PublishArticle 发布文章
func (h *Handler) PublishArticle(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		Error(c, http.StatusBadRequest, "无效的文章ID")
		return
	}

	article, err := h.articleService.PublishArticle(uint(id))
	if err != nil {
		Error(c, http.StatusInternalServerError, "发布文章失败: "+err.Error())
		return
	}

	Success(c, article)
}

// ArchiveArticle 归档文章
func (h *Handler) ArchiveArticle(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		Error(c, http.StatusBadRequest, "无效的文章ID")
		return
	}

	article, err := h.articleService.ArchiveArticle(uint(id))
	if err != nil {
		Error(c, http.StatusInternalServerError, "归档文章失败: "+err.Error())
		return
	}

	Success(c, article)
}

// DeleteArticle 删除文章
func (h *Handler) DeleteArticle(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		Error(c, http.StatusBadRequest, "无效的文章ID")
		return
	}

	if err := h.articleService.DeleteArticle(uint(id)); err != nil {
		Error(c, http.StatusInternalServerError, "删除文章失败: "+err.Error())
		return
	}

	Success(c, nil)
}

// GetSitemap 获取Sitemap
func (h *Handler) GetSitemap(c *gin.Context) {
	// 获取所有已发布的文章
	articles, _, err := h.articleService.GetArticles(1, 1000, nil, "published")
	if err != nil {
		Error(c, http.StatusInternalServerError, "获取文章失败: "+err.Error())
		return
	}

	// 生成Sitemap
	sitemap, err := h.seoService.GenerateSitemap(articles)
	if err != nil {
		Error(c, http.StatusInternalServerError, "生成Sitemap失败: "+err.Error())
		return
	}

	c.Header("Content-Type", "application/xml")
	c.String(http.StatusOK, sitemap)
}

// GetRobotsTxt 获取robots.txt
func (h *Handler) GetRobotsTxt(c *gin.Context) {
	robotsTxt := h.seoService.GenerateRobotsTxt()
	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, robotsTxt)
}

// Register 用户注册
func (h *Handler) Register(c *gin.Context) {
	var req services.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "无效的请求参数")
		return
	}

	user, err := h.authService.Register(req)
	if err != nil {
		Error(c, 400, err.Error())
		return
	}

	Success(c, user)
}

// Login 用户登录
func (h *Handler) Login(c *gin.Context) {
	var req services.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "无效的请求参数")
		return
	}

	token, err := h.authService.Login(req)
	if err != nil {
		Error(c, 401, err.Error())
		return
	}

	Success(c, token)
}

// RefreshToken 刷新令牌
func (h *Handler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "无效的请求参数")
		return
	}

	token, err := h.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		Error(c, 401, err.Error())
		return
	}

	Success(c, token)
}

// GetCurrentUser 获取当前用户信息
func (h *Handler) GetCurrentUser(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		Error(c, 401, "未认证")
		return
	}

	Success(c, user)
}

// BatchGenerateArticles 批量生成文章
func (h *Handler) BatchGenerateArticles(c *gin.Context) {
	var req struct {
		KeywordIDs  []uint `json:"keyword_ids" binding:"required"`
		CategoryIDs []uint `json:"category_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "无效的请求参数: "+err.Error())
		return
	}

	// 获取当前用户ID
	user, exists := c.Get("user")
	if !exists {
		Error(c, http.StatusUnauthorized, "未认证")
		return
	}
	userModel := user.(*models.User)

	// 添加批量任务
	taskIDs, err := h.queueService.BatchAddTasks(c.Request.Context(), req.KeywordIDs, req.CategoryIDs, userModel.ID)
	if err != nil {
		Error(c, http.StatusInternalServerError, "添加生成任务失败: "+err.Error())
		return
	}

	Success(c, gin.H{
		"task_ids": taskIDs,
		"message":  "任务已添加到队列",
	})
}

// GetTaskStatus 获取任务状态
func (h *Handler) GetTaskStatus(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		Error(c, http.StatusBadRequest, "无效的任务ID")
		return
	}

	// 获取当前用户ID
	user, exists := c.Get("user")
	if !exists {
		Error(c, http.StatusUnauthorized, "未认证")
		return
	}
	userModel := user.(*models.User)

	// 获取任务信息
	task, err := h.queueService.GetTask(c.Request.Context(), taskID)
	if err != nil {
		Error(c, http.StatusInternalServerError, "获取任务状态失败: "+err.Error())
		return
	}

	// 验证任务所有权
	if task.UserID != userModel.ID {
		Error(c, http.StatusForbidden, "无权访问此任务")
		return
	}

	Success(c, task)
}

// GetTaskList 获取任务列表
func (h *Handler) GetTaskList(c *gin.Context) {
	// 获取当前用户ID
	user, exists := c.Get("user")
	if !exists {
		Error(c, http.StatusUnauthorized, "未认证")
		return
	}
	userModel := user.(*models.User)

	// 获取任务列表
	tasks, err := h.queueService.GetTaskList(c.Request.Context(), userModel.ID)
	if err != nil {
		Error(c, http.StatusInternalServerError, "获取任务列表失败: "+err.Error())
		return
	}

	Success(c, tasks)
}
