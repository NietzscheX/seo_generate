package services

import (
	"fmt"

	"github.com/NietzscheX/seo-generate/config"
	"github.com/NietzscheX/seo-generate/internal/models"
	"github.com/NietzscheX/seo-generate/pkg/seo"
	"gorm.io/gorm"
)

// KeywordService 关键词服务
type KeywordService struct {
	db            *gorm.DB
	config        *config.Config
	api5118Client *seo.API5118Client
}

// NewKeywordService 创建关键词服务
func NewKeywordService(db *gorm.DB, cfg *config.Config) *KeywordService {
	return &KeywordService{
		db:            db,
		config:        cfg,
		api5118Client: seo.NewAPI5118Client(cfg),
	}
}

// FetchKeywordsByCategory 按分类获取关键词
func (s *KeywordService) FetchKeywordsByCategory(category string, limit int) ([]models.Keyword, error) {
	// 从5118 API获取关键词
	keywords, err := s.api5118Client.GetKeywordsByCategory(category, limit)
	if err != nil {
		return nil, fmt.Errorf("从5118获取关键词失败: %w", err)
	}

	// 清洗关键词
	cleanedKeywords := seo.CleanKeywords(keywords)

	// 保存关键词到数据库
	if err := s.SaveKeywords(cleanedKeywords); err != nil {
		return nil, fmt.Errorf("保存关键词失败: %w", err)
	}

	return cleanedKeywords, nil
}

// SaveKeywords 保存关键词到数据库
func (s *KeywordService) SaveKeywords(keywords []models.Keyword) error {
	// 开始事务
	tx := s.db.Begin()

	for i := range keywords {
		// 检查关键词是否已存在
		var existingKeyword models.Keyword
		result := tx.Where("word = ?", keywords[i].Word).First(&existingKeyword)

		if result.Error == nil {
			// 关键词已存在，更新搜索量
			if keywords[i].SearchVolume > existingKeyword.SearchVolume {
				tx.Model(&existingKeyword).Update("search_volume", keywords[i].SearchVolume)
			}
		} else if result.Error == gorm.ErrRecordNotFound {
			// 关键词不存在，创建新记录
			if err := tx.Create(&keywords[i]).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("创建关键词失败: %w", err)
			}
		} else {
			// 其他错误
			tx.Rollback()
			return fmt.Errorf("查询关键词失败: %w", result.Error)
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	return nil
}

// GetKeywordsByCategory 从数据库获取指定分类的关键词
func (s *KeywordService) GetKeywordsByCategory(categoryID uint, page, pageSize int) ([]models.Keyword, int64, error) {
	var keywords []models.Keyword
	var total int64

	// 查询总数
	query := s.db.Model(&models.Keyword{}).
		Joins("JOIN category_keywords ON category_keywords.keyword_id = keywords.id").
		Where("category_keywords.category_id = ?", categoryID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计关键词数量失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&keywords).Error; err != nil {
		return nil, 0, fmt.Errorf("查询关键词失败: %w", err)
	}

	return keywords, total, nil
}

// GetKeywordByID 根据ID获取关键词
func (s *KeywordService) GetKeywordByID(id uint) (*models.Keyword, error) {
	var keyword models.Keyword
	if err := s.db.First(&keyword, id).Error; err != nil {
		return nil, fmt.Errorf("查询关键词失败: %w", err)
	}
	return &keyword, nil
}

// SearchKeywords 搜索关键词
func (s *KeywordService) SearchKeywords(query string, page, pageSize int) ([]models.Keyword, int64, error) {
	var keywords []models.Keyword
	var total int64

	// 构建查询
	dbQuery := s.db.Model(&models.Keyword{}).Where("word LIKE ?", "%"+query+"%")

	// 统计总数
	if err := dbQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计关键词数量失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := dbQuery.Offset(offset).Limit(pageSize).Find(&keywords).Error; err != nil {
		return nil, 0, fmt.Errorf("搜索关键词失败: %w", err)
	}

	return keywords, total, nil
}

// AssignKeywordToCategory 将关键词分配到分类
func (s *KeywordService) AssignKeywordToCategory(keywordID, categoryID uint) error {
	// 查询关键词
	var keyword models.Keyword
	if err := s.db.First(&keyword, keywordID).Error; err != nil {
		return fmt.Errorf("查询关键词失败: %w", err)
	}

	// 查询分类
	var category models.Category
	if err := s.db.First(&category, categoryID).Error; err != nil {
		return fmt.Errorf("查询分类失败: %w", err)
	}

	// 关联关键词和分类
	if err := s.db.Model(&keyword).Association("Categories").Append(&category); err != nil {
		return fmt.Errorf("关联关键词和分类失败: %w", err)
	}

	return nil
}
