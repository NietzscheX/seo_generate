package services

import (
	"fmt"
	"time"

	"github.com/NietzscheX/seo-generate/internal/models"
	"gorm.io/gorm"
)

// ArticleService 文章服务
type ArticleService struct {
	db *gorm.DB
}

// NewArticleService 创建文章服务
func NewArticleService(db *gorm.DB) *ArticleService {
	return &ArticleService{
		db: db,
	}
}

// GetArticleByID 根据ID获取文章
func (s *ArticleService) GetArticleByID(id uint) (*models.Article, error) {
	var article models.Article
	if err := s.db.Preload("Keywords").Preload("Categories").First(&article, id).Error; err != nil {
		return nil, fmt.Errorf("查询文章失败: %w", err)
	}
	return &article, nil
}

// GetArticleBySlug 根据Slug获取文章
func (s *ArticleService) GetArticleBySlug(slug string) (*models.Article, error) {
	var article models.Article
	if err := s.db.Preload("Keywords").Preload("Categories").Where("slug = ?", slug).First(&article).Error; err != nil {
		return nil, fmt.Errorf("查询文章失败: %w", err)
	}

	// 更新浏览次数
	s.db.Model(&article).Update("view_count", article.ViewCount+1)

	return &article, nil
}

// GetArticles 获取文章列表
func (s *ArticleService) GetArticles(page, pageSize int, categoryID *uint, status string) ([]models.Article, int64, error) {
	var articles []models.Article
	var total int64

	// 构建查询
	query := s.db.Model(&models.Article{})

	// 按状态筛选
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 按分类筛选
	if categoryID != nil {
		query = query.Joins("JOIN category_articles ON category_articles.article_id = articles.id").
			Where("category_articles.category_id = ?", *categoryID)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计文章数量失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Preload("Keywords").Preload("Categories").
		Order("created_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&articles).Error; err != nil {
		return nil, 0, fmt.Errorf("查询文章失败: %w", err)
	}

	return articles, total, nil
}

// SearchArticles 搜索文章
func (s *ArticleService) SearchArticles(query string, page, pageSize int) ([]models.Article, int64, error) {
	var articles []models.Article
	var total int64

	// 构建查询
	dbQuery := s.db.Model(&models.Article{}).
		Where("title LIKE ? OR content LIKE ?", "%"+query+"%", "%"+query+"%")

	// 统计总数
	if err := dbQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计文章数量失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := dbQuery.Preload("Keywords").Preload("Categories").
		Order("created_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&articles).Error; err != nil {
		return nil, 0, fmt.Errorf("搜索文章失败: %w", err)
	}

	return articles, total, nil
}

// UpdateArticle 更新文章
func (s *ArticleService) UpdateArticle(id uint, title, content, summary, metaTitle, metaDesc string, categoryIDs []uint) (*models.Article, error) {
	var article models.Article
	if err := s.db.First(&article, id).Error; err != nil {
		return nil, fmt.Errorf("查询文章失败: %w", err)
	}

	// 开始事务
	tx := s.db.Begin()

	// 更新文章
	article.Title = title
	article.Content = content
	article.Summary = summary
	article.MetaTitle = metaTitle
	article.MetaDesc = metaDesc

	if err := tx.Save(&article).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("更新文章失败: %w", err)
	}

	// 更新分类关联
	if len(categoryIDs) > 0 {
		// 清除现有关联
		if err := tx.Model(&article).Association("Categories").Clear(); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("清除分类关联失败: %w", err)
		}

		// 添加新关联
		var categories []models.Category
		if err := tx.Where("id IN ?", categoryIDs).Find(&categories).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("查询分类失败: %w", err)
		}

		if err := tx.Model(&article).Association("Categories").Append(categories); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("关联分类失败: %w", err)
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}

	return &article, nil
}

// PublishArticle 发布文章
func (s *ArticleService) PublishArticle(id uint) (*models.Article, error) {
	var article models.Article
	if err := s.db.First(&article, id).Error; err != nil {
		return nil, fmt.Errorf("查询文章失败: %w", err)
	}

	// 设置发布状态和时间
	now := time.Now()
	article.Status = "published"
	article.PublishedAt = &now

	if err := s.db.Save(&article).Error; err != nil {
		return nil, fmt.Errorf("发布文章失败: %w", err)
	}

	return &article, nil
}

// ArchiveArticle 归档文章
func (s *ArticleService) ArchiveArticle(id uint) (*models.Article, error) {
	var article models.Article
	if err := s.db.First(&article, id).Error; err != nil {
		return nil, fmt.Errorf("查询文章失败: %w", err)
	}

	// 设置归档状态
	article.Status = "archived"

	if err := s.db.Save(&article).Error; err != nil {
		return nil, fmt.Errorf("归档文章失败: %w", err)
	}

	return &article, nil
}

// DeleteArticle 删除文章
func (s *ArticleService) DeleteArticle(id uint) error {
	// 开始事务
	tx := s.db.Begin()

	// 删除文章与关键词的关联
	if err := tx.Exec("DELETE FROM keyword_articles WHERE article_id = ?", id).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除文章与关键词的关联失败: %w", err)
	}

	// 删除文章与分类的关联
	if err := tx.Exec("DELETE FROM category_articles WHERE article_id = ?", id).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除文章与分类的关联失败: %w", err)
	}

	// 删除文章
	if err := tx.Delete(&models.Article{}, id).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除文章失败: %w", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	return nil
}

// GetRelatedArticles 获取相关文章
func (s *ArticleService) GetRelatedArticles(articleID uint, limit int) ([]models.Article, error) {
	var article models.Article
	if err := s.db.Preload("Keywords").First(&article, articleID).Error; err != nil {
		return nil, fmt.Errorf("查询文章失败: %w", err)
	}

	// 如果文章没有关键词，返回最新文章
	if len(article.Keywords) == 0 {
		var articles []models.Article
		if err := s.db.Where("id != ? AND status = ?", articleID, "published").
			Order("published_at DESC").
			Limit(limit).
			Find(&articles).Error; err != nil {
			return nil, fmt.Errorf("查询最新文章失败: %w", err)
		}
		return articles, nil
	}

	// 提取关键词ID
	keywordIDs := make([]uint, len(article.Keywords))
	for i, kw := range article.Keywords {
		keywordIDs[i] = kw.ID
	}

	// 查询具有相同关键词的文章
	var articles []models.Article
	if err := s.db.Distinct("articles.*").
		Joins("JOIN keyword_articles ON keyword_articles.article_id = articles.id").
		Where("articles.id != ? AND articles.status = ? AND keyword_articles.keyword_id IN ?",
			articleID, "published", keywordIDs).
		Order("articles.published_at DESC").
		Limit(limit).
		Find(&articles).Error; err != nil {
		return nil, fmt.Errorf("查询相关文章失败: %w", err)
	}

	return articles, nil
}
