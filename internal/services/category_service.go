package services

import (
	"fmt"

	"github.com/NietzscheX/seo-generate/internal/models"
	"gorm.io/gorm"
)

// CategoryService 分类服务
type CategoryService struct {
	db *gorm.DB
}

// NewCategoryService 创建分类服务
func NewCategoryService(db *gorm.DB) *CategoryService {
	return &CategoryService{
		db: db,
	}
}

// CreateCategory 创建分类
func (s *CategoryService) CreateCategory(name string, parentID *uint) (*models.Category, error) {
	category := models.Category{
		Name:     name,
		ParentID: parentID,
	}

	if err := s.db.Create(&category).Error; err != nil {
		return nil, fmt.Errorf("创建分类失败: %w", err)
	}

	return &category, nil
}

// GetCategoryByID 根据ID获取分类
func (s *CategoryService) GetCategoryByID(id uint) (*models.Category, error) {
	var category models.Category
	if err := s.db.First(&category, id).Error; err != nil {
		return nil, fmt.Errorf("查询分类失败: %w", err)
	}
	return &category, nil
}

// GetCategoryWithChildren 获取分类及其子分类
func (s *CategoryService) GetCategoryWithChildren(id uint) (*models.Category, error) {
	var category models.Category
	if err := s.db.Preload("Children").First(&category, id).Error; err != nil {
		return nil, fmt.Errorf("查询分类失败: %w", err)
	}
	return &category, nil
}

// GetAllCategories 获取所有分类
func (s *CategoryService) GetAllCategories() ([]models.Category, error) {
	var categories []models.Category
	if err := s.db.Find(&categories).Error; err != nil {
		return nil, fmt.Errorf("查询所有分类失败: %w", err)
	}
	return categories, nil
}

// GetRootCategories 获取所有根分类
func (s *CategoryService) GetRootCategories() ([]models.Category, error) {
	var categories []models.Category
	if err := s.db.Where("parent_id IS NULL").Find(&categories).Error; err != nil {
		return nil, fmt.Errorf("查询根分类失败: %w", err)
	}
	return categories, nil
}

// GetCategoryTree 获取分类树
func (s *CategoryService) GetCategoryTree() ([]models.Category, error) {
	var rootCategories []models.Category
	if err := s.db.Preload("Children").Where("parent_id IS NULL").Find(&rootCategories).Error; err != nil {
		return nil, fmt.Errorf("查询分类树失败: %w", err)
	}

	// 递归加载子分类的子分类
	for i := range rootCategories {
		if err := s.loadChildrenRecursive(&rootCategories[i]); err != nil {
			return nil, err
		}
	}

	return rootCategories, nil
}

// loadChildrenRecursive 递归加载子分类
func (s *CategoryService) loadChildrenRecursive(category *models.Category) error {
	if len(category.Children) == 0 {
		return nil
	}

	for i := range category.Children {
		if err := s.db.Preload("Children").First(&category.Children[i], category.Children[i].ID).Error; err != nil {
			return fmt.Errorf("加载子分类失败: %w", err)
		}

		if err := s.loadChildrenRecursive(&category.Children[i]); err != nil {
			return err
		}
	}

	return nil
}

// UpdateCategory 更新分类
func (s *CategoryService) UpdateCategory(id uint, name string, parentID *uint) (*models.Category, error) {
	var category models.Category
	if err := s.db.First(&category, id).Error; err != nil {
		return nil, fmt.Errorf("查询分类失败: %w", err)
	}

	// 检查是否将分类设为自己的子分类
	if parentID != nil && *parentID == id {
		return nil, fmt.Errorf("不能将分类设为自己的子分类")
	}

	// 检查是否将分类设为其子分类的子分类
	if parentID != nil {
		var children []models.Category
		if err := s.db.Where("parent_id = ?", id).Find(&children).Error; err != nil {
			return nil, fmt.Errorf("查询子分类失败: %w", err)
		}

		for _, child := range children {
			if child.ID == *parentID {
				return nil, fmt.Errorf("不能将分类设为其子分类的子分类")
			}
		}
	}

	// 更新分类
	category.Name = name
	category.ParentID = parentID

	if err := s.db.Save(&category).Error; err != nil {
		return nil, fmt.Errorf("更新分类失败: %w", err)
	}

	return &category, nil
}

// DeleteCategory 删除分类
func (s *CategoryService) DeleteCategory(id uint) error {
	// 检查是否有子分类
	var childCount int64
	if err := s.db.Model(&models.Category{}).Where("parent_id = ?", id).Count(&childCount).Error; err != nil {
		return fmt.Errorf("检查子分类失败: %w", err)
	}

	if childCount > 0 {
		return fmt.Errorf("不能删除有子分类的分类")
	}

	// 开始事务
	tx := s.db.Begin()

	// 删除分类与关键词的关联
	if err := tx.Exec("DELETE FROM category_keywords WHERE category_id = ?", id).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除分类与关键词的关联失败: %w", err)
	}

	// 删除分类与文章的关联
	if err := tx.Exec("DELETE FROM category_articles WHERE category_id = ?", id).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除分类与文章的关联失败: %w", err)
	}

	// 删除分类
	if err := tx.Delete(&models.Category{}, id).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除分类失败: %w", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	return nil
}

// InitDefaultCategories 初始化默认分类
func (s *CategoryService) InitDefaultCategories() error {
	// 检查是否已有分类
	var count int64
	if err := s.db.Model(&models.Category{}).Count(&count).Error; err != nil {
		return fmt.Errorf("检查分类数量失败: %w", err)
	}

	if count > 0 {
		return nil // 已有分类，不需要初始化
	}

	// 定义默认分类
	defaultCategories := []struct {
		Name     string
		Children []string
	}{
		{
			Name: "中医理论",
			Children: []string{
				"阴阳五行",
				"脏腑经络",
				"气血津液",
				"病因病机",
			},
		},
		{
			Name: "养生方法",
			Children: []string{
				"饮食养生",
				"运动养生",
				"起居养生",
				"情志养生",
				"四季养生",
			},
		},
		{
			Name: "修行技巧",
			Children: []string{
				"冥想打坐",
				"气功导引",
				"太极拳法",
				"心性修炼",
			},
		},
	}

	// 开始事务
	tx := s.db.Begin()

	// 创建分类
	for _, cat := range defaultCategories {
		parentCat := models.Category{
			Name: cat.Name,
		}

		if err := tx.Create(&parentCat).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("创建父分类失败: %w", err)
		}

		for _, childName := range cat.Children {
			childCat := models.Category{
				Name:     childName,
				ParentID: &parentCat.ID,
			}

			if err := tx.Create(&childCat).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("创建子分类失败: %w", err)
			}
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	return nil
}
