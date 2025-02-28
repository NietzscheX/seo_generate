package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/NietzscheX/seo-generate/config"
	"github.com/NietzscheX/seo-generate/internal/models"
)

// OllamaClient Ollama API客户端
type OllamaClient struct {
	config     *config.Config
	httpClient *http.Client
}

// NewOllamaClient 创建Ollama API客户端
func NewOllamaClient(cfg *config.Config) *OllamaClient {
	return &OllamaClient{
		config: cfg,
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.AI.Timeout) * time.Second,
		},
	}
}

// OllamaRequest Ollama请求
type OllamaRequest struct {
	Model       string  `json:"model"`
	Prompt      string  `json:"prompt"`
	System      string  `json:"system"`
	Temperature float64 `json:"temperature"`
	Stream      bool    `json:"stream"`
}

// OllamaResponse Ollama响应
type OllamaResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
}

// GenerateContent 生成内容
func (c *OllamaClient) GenerateContent(ctx context.Context, prompt string) (string, error) {
	url := fmt.Sprintf("%s/generate", c.config.AI.OllamaEndpoint)

	// 构建请求体
	requestBody, err := json.Marshal(OllamaRequest{
		Model:       "llama3", // 使用默认模型，可以根据需要修改
		Prompt:      prompt,
		System:      "你是一个专业的内容创作者，擅长撰写养生、中医和修行相关的高质量文章。请根据用户提供的关键词和要求，创作SEO友好的内容。",
		Temperature: c.config.AI.Temperature,
		Stream:      false,
	})
	if err != nil {
		return "", fmt.Errorf("序列化请求体失败: %w", err)
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	duration := time.Since(startTime).Milliseconds()

	// 记录API调用日志
	apiLog := models.APILog{
		APIName:   "ollama",
		Endpoint:  url,
		Request:   string(requestBody),
		Duration:  int(duration),
		CreatedAt: time.Now(),
	}

	if err != nil {
		apiLog.Status = 0
		apiLog.Response = err.Error()
		// 保存日志
		return "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		apiLog.Status = resp.StatusCode
		apiLog.Response = err.Error()
		// 保存日志
		return "", fmt.Errorf("读取响应体失败: %w", err)
	}

	apiLog.Status = resp.StatusCode
	apiLog.Response = string(respBody)
	// 保存日志

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API错误: %s", string(respBody))
	}

	// 解析响应
	var response OllamaResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	return response.Response, nil
}

// StreamGenerateContent 流式生成内容
func (c *OllamaClient) StreamGenerateContent(ctx context.Context, prompt string, contentChan chan<- string, errorChan chan<- error) {
	url := fmt.Sprintf("%s/generate", c.config.AI.OllamaEndpoint)

	// 构建请求体
	requestBody, err := json.Marshal(OllamaRequest{
		Model:       "llama3", // 使用默认模型，可以根据需要修改
		Prompt:      prompt,
		System:      "你是一个专业的内容创作者，擅长撰写养生、中医和修行相关的高质量文章。请根据用户提供的关键词和要求，创作SEO友好的内容。",
		Temperature: c.config.AI.Temperature,
		Stream:      true,
	})
	if err != nil {
		errorChan <- fmt.Errorf("序列化请求体失败: %w", err)
		return
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		errorChan <- fmt.Errorf("创建请求失败: %w", err)
		return
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	duration := time.Since(startTime).Milliseconds()

	// 记录API调用日志
	apiLog := models.APILog{
		APIName:   "ollama_stream",
		Endpoint:  url,
		Request:   string(requestBody),
		Duration:  int(duration),
		CreatedAt: time.Now(),
	}

	if err != nil {
		apiLog.Status = 0
		apiLog.Response = err.Error()
		// 保存日志
		errorChan <- fmt.Errorf("请求失败: %w", err)
		return
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		apiLog.Status = resp.StatusCode
		apiLog.Response = string(respBody)
		// 保存日志
		errorChan <- fmt.Errorf("API错误: %s", string(respBody))
		return
	}

	// 读取流式响应
	decoder := json.NewDecoder(resp.Body)
	for {
		var streamResp OllamaResponse
		if err := decoder.Decode(&streamResp); err != nil {
			if err == io.EOF {
				break
			}
			errorChan <- fmt.Errorf("解析流式响应失败: %w", err)
			return
		}

		// 发送内容
		if streamResp.Response != "" {
			contentChan <- streamResp.Response
		}

		// 检查是否完成
		if streamResp.Done {
			break
		}
	}

	// 完成
	close(contentChan)
}
