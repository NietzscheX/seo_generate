package seo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/NietzscheX/seo-generate/config"
	"github.com/NietzscheX/seo-generate/internal/models"
)

// API5118Client 5118 API客户端
type API5118Client struct {
	config     *config.Config
	httpClient *http.Client
}

// NewAPI5118Client 创建5118 API客户端
func NewAPI5118Client(cfg *config.Config) *API5118Client {
	return &API5118Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout: time.Second * 30,
		},
	}
}

// KeywordSearchResponse 关键词搜索响应
type KeywordSearchResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Total int `json:"total"`
		Items []struct {
			Keyword      string `json:"keyword"`
			SearchVolume int    `json:"search_volume"`
		} `json:"items"`
	} `json:"data"`
}

// SearchKeywords 搜索关键词
func (c *API5118Client) SearchKeywords(query string, page, pageSize int) ([]models.Keyword, int, error) {
	url := fmt.Sprintf("%s/keyword/search", c.config.API5118.BaseURL)

	// 构建请求体
	requestBody, err := json.Marshal(map[string]interface{}{
		"query":     query,
		"page":      page,
		"page_size": pageSize,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("序列化请求体失败: %w", err)
	}

	// 创建请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, 0, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.API5118.Key))

	// 发送请求
	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	duration := time.Since(startTime).Milliseconds()

	// 记录API调用日志
	apiLog := models.APILog{
		APIName:   "5118",
		Endpoint:  url,
		Request:   string(requestBody),
		Duration:  int(duration),
		CreatedAt: time.Now(),
	}

	if err != nil {
		apiLog.Status = 0
		apiLog.Response = err.Error()
		// 这里应该保存日志到数据库，但为简化示例，暂不实现
		return nil, 0, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		apiLog.Status = resp.StatusCode
		apiLog.Response = err.Error()
		// 保存日志
		return nil, 0, fmt.Errorf("读取响应体失败: %w", err)
	}

	apiLog.Status = resp.StatusCode
	apiLog.Response = string(respBody)
	// 保存日志

	// 解析响应
	var response KeywordSearchResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, 0, fmt.Errorf("解析响应失败: %w", err)
	}

	// 检查响应状态
	if response.Code != 200 {
		return nil, 0, fmt.Errorf("API错误: %s", response.Message)
	}

	// 转换为关键词模型
	keywords := make([]models.Keyword, 0, len(response.Data.Items))
	for _, item := range response.Data.Items {
		keywords = append(keywords, models.Keyword{
			Word:         item.Keyword,
			SearchVolume: item.SearchVolume,
			Source:       "5118",
			Status:       "active",
		})
	}

	return keywords, response.Data.Total, nil
}

// GetKeywordsByCategory 按分类获取关键词
func (c *API5118Client) GetKeywordsByCategory(category string, limit int) ([]models.Keyword, error) {
	// 计算需要请求的页数
	pageSize := 100 // 5118 API每页最大100条
	totalPages := int(math.Ceil(float64(limit) / float64(pageSize)))

	allKeywords := make([]models.Keyword, 0, limit)
	var total int

	// 分页请求关键词
	for page := 1; page <= totalPages; page++ {
		// 计算当前页需要获取的数量
		currentPageSize := pageSize
		if page == totalPages {
			remaining := limit - (page-1)*pageSize
			if remaining < pageSize {
				currentPageSize = remaining
			}
		}

		// 搜索关键词
		keywords, newTotal, err := c.SearchKeywords(category, page, currentPageSize)
		if err != nil {
			return nil, err
		}

		total = newTotal
		allKeywords = append(allKeywords, keywords...)

		// 如果已获取全部关键词，提前结束
		if len(allKeywords) >= limit || len(allKeywords) >= total {
			break
		}

		// 避免请求过快
		time.Sleep(time.Second)
	}

	// 限制返回数量
	if len(allKeywords) > limit {
		allKeywords = allKeywords[:limit]
	}

	return allKeywords, nil
}

// CleanKeywords 清洗关键词
func CleanKeywords(keywords []models.Keyword) []models.Keyword {
	// 去重映射
	uniqueMap := make(map[string]models.Keyword)

	for _, kw := range keywords {
		// 去除无效字符（这里可以根据需要添加更多过滤规则）
		word := kw.Word

		// 如果关键词不为空且长度合适，则保留
		if word != "" && len(word) >= 2 && len(word) <= 50 {
			// 如果已存在相同关键词，保留搜索量更高的
			if existing, ok := uniqueMap[word]; ok {
				if kw.SearchVolume > existing.SearchVolume {
					uniqueMap[word] = kw
				}
			} else {
				uniqueMap[word] = kw
			}
		}
	}

	// 转换回切片
	cleanedKeywords := make([]models.Keyword, 0, len(uniqueMap))
	for _, kw := range uniqueMap {
		cleanedKeywords = append(cleanedKeywords, kw)
	}

	return cleanedKeywords
}
