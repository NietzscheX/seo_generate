package services

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/NietzscheX/seo-generate/config"
	"github.com/NietzscheX/seo-generate/internal/models"
	"github.com/NietzscheX/seo-generate/pkg/ai"
	"gorm.io/gorm"
)

// ContentService 内容生成服务
type ContentService struct {
	db             *gorm.DB
	config         *config.Config
	deepseekClient *ai.DeepSeekClient
	ollamaClient   *ai.OllamaClient
}

// NewContentService 创建内容生成服务
func NewContentService(db *gorm.DB, cfg *config.Config) *ContentService {
	return &ContentService{
		db:             db,
		config:         cfg,
		deepseekClient: ai.NewDeepSeekClient(cfg),
		ollamaClient:   ai.NewOllamaClient(cfg),
	}
}

// PromptTemplate 提示模板
const PromptTemplate = `
请根据以下关键词，创作一篇关于养生/中医/修行的高质量文章：

主要关键词: %s

文章要求:
1. 标题需要包含主关键词，吸引人点击
2. 内容长度在%d-%d字之间
3. 分段清晰，每段不超过300字
4. 使用二级标题(##)和三级标题(###)组织内容
5. 内容需要专业、准确、有深度
6. 适当引用中医经典或科学研究支持观点
7. 结尾要有总结和实用建议

文章格式:
- 使用Markdown格式
- 标题使用一级标题(#)
- 正文分段使用空行隔开
- 重要概念可以使用**加粗**标记
- 可以适当使用列表展示步骤或要点

请确保内容原创、有价值，避免虚假或误导性信息。
`

// GenerateArticle 生成文章
func (s *ContentService) GenerateArticle(ctx context.Context, keyword models.Keyword, categoryIDs []uint) (*models.Article, error) {
	// 创建生成任务
	task := models.GenerationTask{
		KeywordID: keyword.ID,
		Status:    "processing",
		Prompt:    fmt.Sprintf(PromptTemplate, keyword.Word, s.config.Content.ArticleMinLength, s.config.Content.ArticleMaxLength),
	}

	if err := s.db.Create(&task).Error; err != nil {
		return nil, fmt.Errorf("创建生成任务失败: %w", err)
	}

	// 尝试使用DeepSeek生成内容
	content, err := s.generateWithDeepSeek(ctx, task.Prompt)
	if err != nil {
		// 如果DeepSeek失败，尝试使用Ollama
		s.db.Model(&task).Updates(map[string]interface{}{
			"error_message": err.Error(),
		})

		content, err = s.generateWithOllama(ctx, task.Prompt)
		if err != nil {
			// 更新任务状态为失败
			s.db.Model(&task).Updates(map[string]interface{}{
				"status":        "failed",
				"error_message": err.Error(),
			})
			return nil, fmt.Errorf("生成内容失败: %w", err)
		}

		// 更新使用的模型
		s.db.Model(&task).Update("model_used", "ollama")
	} else {
		// 更新使用的模型
		s.db.Model(&task).Update("model_used", "deepseek")
	}

	// 打印原始内容
	fmt.Println("=== 原始AI生成内容 ===")
	fmt.Println(content)
	fmt.Println("=== 原始内容结束 ===")

	// 过滤内容
	content = ai.FilterContent(content)

	// 打印过滤后的内容
	fmt.Println("=== 过滤后的内容 ===")
	fmt.Println(content)
	fmt.Println("=== 过滤后内容结束 ===")

	// 解析标题和内容
	title, content := parseArticle(content)

	// 打印解析后的标题和内容
	fmt.Println("=== 解析后的标题 ===")
	fmt.Println(title)
	fmt.Println("=== 解析后的内容 ===")
	fmt.Println(content)

	// 生成摘要
	summary := generateSummary(content)
	fmt.Println("=== 生成的摘要 ===")
	fmt.Println(summary)

	// 生成slug
	slug := generateSlug(title)
	fmt.Println("=== 生成的Slug ===")
	fmt.Println(slug)

	// 在保存到数据库前清理内容
	title = strings.Map(func(r rune) rune {
		if r < 32 || r > 126 && r < 256 {
			return -1
		}
		return r
	}, title)

	content = strings.Map(func(r rune) rune {
		if r < 32 || r > 126 && r < 256 {
			return -1
		}
		return r
	}, content)

	summary = strings.Map(func(r rune) rune {
		if r < 32 || r > 126 && r < 256 {
			return -1
		}
		return r
	}, summary)

	// 打印清理后的内容
	fmt.Println("=== 清理后的标题 ===")
	fmt.Println(title)
	fmt.Println("=== 清理后的内容 ===")
	fmt.Println(content)
	fmt.Println("=== 清理后的摘要 ===")
	fmt.Println(summary)

	// 创建文章
	article := &models.Article{
		Title:     title,
		Slug:      slug,
		Content:   content,
		Summary:   summary,
		MetaTitle: title,
		MetaDesc:  summary[:min(len(summary), 160)],
		Status:    "draft",
	}

	// 开始事务
	tx := s.db.Begin()

	// 创建文章
	if err := tx.Create(article).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("创建文章失败: %w", err)
	}

	// 关联关键词
	if err := tx.Model(article).Association("Keywords").Append(&keyword); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("关联关键词失败: %w", err)
	}

	// 关联分类
	if len(categoryIDs) > 0 {
		var categories []models.Category
		if err := tx.Where("id IN ?", categoryIDs).Find(&categories).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("查询分类失败: %w", err)
		}

		if err := tx.Model(article).Association("Categories").Append(categories); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("关联分类失败: %w", err)
		}
	}

	// 更新任务状态
	if err := tx.Model(&task).Updates(map[string]interface{}{
		"status":     "completed",
		"article_id": article.ID,
	}).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("更新任务状态失败: %w", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}

	return article, nil
}

// generateWithDeepSeek 使用DeepSeek生成内容
func (s *ContentService) generateWithDeepSeek(ctx context.Context, prompt string) (string, error) {
	// 创建超时上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(s.config.AI.Timeout)*time.Second)
	defer cancel()

	return s.deepseekClient.GenerateContent(timeoutCtx, prompt)
}

// generateWithOllama 使用Ollama生成内容
func (s *ContentService) generateWithOllama(ctx context.Context, prompt string) (string, error) {
	// 创建超时上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(s.config.AI.Timeout)*time.Second)
	defer cancel()

	return s.ollamaClient.GenerateContent(timeoutCtx, prompt)
}

// parseArticle 解析文章标题和内容
func parseArticle(content string) (string, string) {
	// 确保内容是有效的UTF-8
	if !utf8.ValidString(content) {
		content = strings.ToValidUTF8(content, "")
	}

	lines := strings.Split(content, "\n")

	var title string
	var contentLines []string
	var contentStarted bool

	for i, line := range lines {
		line = strings.TrimSpace(line)

		// 跳过空行，除非内容已经开始
		if line == "" {
			if contentStarted {
				contentLines = append(contentLines, "")
			}
			continue
		}

		// 查找标题（Markdown格式）
		if strings.HasPrefix(line, "# ") {
			title = strings.TrimPrefix(line, "# ")
			contentStarted = true
			continue
		}

		// 如果还没找到标题，且当前行不为空，则将其作为标题
		if title == "" && line != "" {
			title = line
			contentStarted = true
			continue
		}

		// 收集内容
		contentStarted = true
		contentLines = append(contentLines, lines[i])
	}

	// 如果没有找到标题，使用默认标题
	if title == "" {
		title = "养生健康文章"
	}

	// 清理标题中的特殊字符
	title = strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) && !unicode.IsControl(r) {
			return r
		}
		return -1
	}, title)

	// 限制标题长度
	if len(title) > 100 {
		title = title[:100]
	}

	// 合并内容行
	content = strings.Join(contentLines, "\n")

	return title, content
}

// generateSummary 生成摘要
func generateSummary(content string) string {
	// 简单实现：取前300个字符作为摘要
	content = strings.ReplaceAll(content, "\n", " ")
	if len(content) > 300 {
		return content[:300] + "..."
	}
	return content
}

// generateSlug 生成slug
func generateSlug(title string) string {
	// 确保标题是有效的UTF-8
	if !utf8.ValidString(title) {
		title = strings.ToValidUTF8(title, "")
	}

	// 将标题转换为小写
	slug := strings.ToLower(title)

	// 替换中文字符为拼音（简化处理，实际应用中可能需要更复杂的拼音转换）
	// 这里简单处理：移除所有非ASCII字符
	slug = strings.Map(func(r rune) rune {
		if r < 128 {
			return r
		}
		return -1
	}, slug)

	// 替换空格为连字符
	slug = strings.ReplaceAll(slug, " ", "-")

	// 移除特殊字符，只保留字母、数字和连字符
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1
	}, slug)

	// 移除多余的连字符
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}

	// 移除首尾的连字符
	slug = strings.Trim(slug, "-")

	// 确保slug不为空
	if slug == "" {
		// 使用时间戳作为slug
		slug = fmt.Sprintf("article-%d", time.Now().Unix())
	}

	// 限制slug长度
	if len(slug) > 100 {
		slug = slug[:100]
		// 确保不以连字符结尾
		slug = strings.TrimRight(slug, "-")
	}

	return slug
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
