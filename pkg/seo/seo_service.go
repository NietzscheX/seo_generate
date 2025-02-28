package seo

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/NietzscheX/seo-generate/config"
	"github.com/NietzscheX/seo-generate/internal/models"
)

// SEOService SEO服务
type SEOService struct {
	config *config.Config
}

// NewSEOService 创建SEO服务
func NewSEOService(cfg *config.Config) *SEOService {
	return &SEOService{
		config: cfg,
	}
}

// GenerateMetaDescription 生成Meta描述
func (s *SEOService) GenerateMetaDescription(content string, maxLength int) string {
	// 清理内容
	content = strings.ReplaceAll(content, "\n", " ")
	content = strings.ReplaceAll(content, "  ", " ")

	// 截取指定长度
	if len(content) > maxLength {
		return content[:maxLength] + "..."
	}
	return content
}

// GenerateCanonicalURL 生成规范URL
func (s *SEOService) GenerateCanonicalURL(slug string) string {
	return fmt.Sprintf("%s/health/%s", s.config.SEO.SiteURL, slug)
}

// ArticleSchema 文章结构化数据
type ArticleSchema struct {
	Context     string `json:"@context"`
	Type        string `json:"@type"`
	Headline    string `json:"headline"`
	Description string `json:"description"`
	Author      struct {
		Type string `json:"@type"`
		Name string `json:"name"`
	} `json:"author"`
	Publisher struct {
		Type string `json:"@type"`
		Name string `json:"name"`
		Logo struct {
			Type string `json:"@type"`
			URL  string `json:"url"`
		} `json:"logo"`
	} `json:"publisher"`
	DatePublished    string `json:"datePublished"`
	DateModified     string `json:"dateModified"`
	MainEntityOfPage struct {
		Type string `json:"@type"`
		ID   string `json:"@id"`
	} `json:"mainEntityOfPage"`
	Image string `json:"image,omitempty"`
}

// GenerateArticleSchema 生成文章结构化数据
func (s *SEOService) GenerateArticleSchema(article *models.Article) *ArticleSchema {
	publishedAt := time.Now().Format(time.RFC3339)
	if article.PublishedAt != nil {
		publishedAt = article.PublishedAt.Format(time.RFC3339)
	}

	schema := &ArticleSchema{
		Context:     "https://schema.org",
		Type:        "Article",
		Headline:    article.Title,
		Description: article.MetaDesc,
		Author: struct {
			Type string `json:"@type"`
			Name string `json:"name"`
		}{
			Type: "Organization",
			Name: s.config.SEO.SiteName,
		},
		Publisher: struct {
			Type string `json:"@type"`
			Name string `json:"name"`
			Logo struct {
				Type string `json:"@type"`
				URL  string `json:"url"`
			} `json:"logo"`
		}{
			Type: "Organization",
			Name: s.config.SEO.SiteName,
			Logo: struct {
				Type string `json:"@type"`
				URL  string `json:"url"`
			}{
				Type: "ImageObject",
				URL:  fmt.Sprintf("%s/static/images/logo.png", s.config.SEO.SiteURL),
			},
		},
		DatePublished: publishedAt,
		DateModified:  article.UpdatedAt.Format(time.RFC3339),
		MainEntityOfPage: struct {
			Type string `json:"@type"`
			ID   string `json:"@id"`
		}{
			Type: "WebPage",
			ID:   s.GenerateCanonicalURL(article.Slug),
		},
	}

	return schema
}

// URLSet XML Sitemap URL集合
type URLSet struct {
	XMLName xml.Name `xml:"urlset"`
	XMLNS   string   `xml:"xmlns,attr"`
	URLs    []URL    `xml:"url"`
}

// URL XML Sitemap URL
type URL struct {
	Loc        string  `xml:"loc"`
	LastMod    string  `xml:"lastmod"`
	ChangeFreq string  `xml:"changefreq"`
	Priority   float64 `xml:"priority"`
}

// GenerateSitemap 生成Sitemap
func (s *SEOService) GenerateSitemap(articles []models.Article) (string, error) {
	urlSet := URLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
	}

	// 添加首页
	urlSet.URLs = append(urlSet.URLs, URL{
		Loc:        s.config.SEO.SiteURL,
		LastMod:    time.Now().Format("2006-01-02"),
		ChangeFreq: "daily",
		Priority:   1.0,
	})

	// 添加文章页
	for _, article := range articles {
		lastMod := time.Now().Format("2006-01-02")
		if article.UpdatedAt.After(time.Time{}) {
			lastMod = article.UpdatedAt.Format("2006-01-02")
		}

		urlSet.URLs = append(urlSet.URLs, URL{
			Loc:        s.GenerateCanonicalURL(article.Slug),
			LastMod:    lastMod,
			ChangeFreq: "weekly",
			Priority:   0.8,
		})
	}

	// 生成XML
	output, err := xml.MarshalIndent(urlSet, "", "  ")
	if err != nil {
		return "", fmt.Errorf("生成Sitemap XML失败: %w", err)
	}

	return xml.Header + string(output), nil
}

// GenerateRobotsTxt 生成robots.txt
func (s *SEOService) GenerateRobotsTxt() string {
	return fmt.Sprintf(`User-agent: *
Allow: /
Sitemap: %s/sitemap.xml
`, s.config.SEO.SiteURL)
}
