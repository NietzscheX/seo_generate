package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/NietzscheX/seo-generate/config"
	"github.com/NietzscheX/seo-generate/internal/models"
)

// DeepSeekClient DeepSeek API客户端
type DeepSeekClient struct {
	config     *config.Config
	httpClient *http.Client
}

// NewDeepSeekClient 创建DeepSeek API客户端
func NewDeepSeekClient(cfg *config.Config) *DeepSeekClient {
	return &DeepSeekClient{
		config: cfg,
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.AI.Timeout) * time.Second,
		},
	}
}

// ChatCompletionRequest 聊天完成请求
type ChatCompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
	Stream      bool      `json:"stream"`
}

// Message 聊天消息
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionResponse 聊天完成响应
type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int     `json:"index"`
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// StreamCompletionResponse 流式完成响应
type StreamCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int    `json:"index"`
		Delta        Delta  `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

// Delta 增量内容
type Delta struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// GenerateContent 生成内容
func (c *DeepSeekClient) GenerateContent(ctx context.Context, prompt string) (string, error) {
	url := fmt.Sprintf("%s/chat/completions", c.config.AI.DeepseekAPIURL)

	// 构建请求体
	requestBody, err := json.Marshal(ChatCompletionRequest{
		Model: "deepseek-chat",
		Messages: []Message{
			{
				Role:    "system",
				Content: "你是一个专业的内容创作者，擅长撰写养生、中医和修行相关的高质量文章。请根据用户提供的关键词和要求，创作SEO友好的内容。",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: c.config.AI.Temperature,
		MaxTokens:   c.config.AI.MaxTokens,
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
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.AI.DeepseekAPIKey))

	// 发送请求
	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	duration := time.Since(startTime).Milliseconds()

	// 记录API调用日志
	apiLog := models.APILog{
		APIName:   "deepseek",
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
	var response ChatCompletionResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	// 检查是否有内容返回
	if len(response.Choices) == 0 {
		return "", fmt.Errorf("没有内容返回")
	}

	return response.Choices[0].Message.Content, nil
}

// StreamGenerateContent 流式生成内容
func (c *DeepSeekClient) StreamGenerateContent(ctx context.Context, prompt string, contentChan chan<- string, errorChan chan<- error) {
	url := fmt.Sprintf("%s/chat/completions", c.config.AI.DeepseekAPIURL)

	// 构建请求体
	requestBody, err := json.Marshal(ChatCompletionRequest{
		Model: "deepseek-chat",
		Messages: []Message{
			{
				Role:    "system",
				Content: "你是一个专业的内容创作者，擅长撰写养生、中医和修行相关的高质量文章。请根据用户提供的关键词和要求，创作SEO友好的内容。",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: c.config.AI.Temperature,
		MaxTokens:   c.config.AI.MaxTokens,
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
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.AI.DeepseekAPIKey))

	// 发送请求
	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	duration := time.Since(startTime).Milliseconds()

	// 记录API调用日志
	apiLog := models.APILog{
		APIName:   "deepseek_stream",
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
	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			errorChan <- fmt.Errorf("读取流式响应失败: %w", err)
			return
		}

		// 跳过空行
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		// 跳过SSE前缀
		if bytes.HasPrefix(line, []byte("data: ")) {
			line = bytes.TrimPrefix(line, []byte("data: "))
		}

		// 检查是否是[DONE]消息
		if string(line) == "[DONE]" {
			break
		}

		// 解析JSON
		var streamResp StreamCompletionResponse
		if err := json.Unmarshal(line, &streamResp); err != nil {
			errorChan <- fmt.Errorf("解析流式响应失败: %w", err)
			return
		}

		// 检查是否有内容
		if len(streamResp.Choices) > 0 && streamResp.Choices[0].Delta.Content != "" {
			contentChan <- streamResp.Choices[0].Delta.Content
		}

		// 检查是否完成
		if len(streamResp.Choices) > 0 && streamResp.Choices[0].FinishReason != "" {
			break
		}
	}

	// 完成
	close(contentChan)
}

// FilterContent 内容安全过滤
func FilterContent(content string) string {
	// 这里可以实现内容安全过滤逻辑
	// 例如：过滤敏感词、违禁内容等

	// 移除不可打印字符
	var result strings.Builder
	for _, r := range content {
		if unicode.IsPrint(r) || r == '\n' || r == '\t' {
			result.WriteRune(r)
		} else {
			// 替换不可打印字符为空格
			result.WriteRune(' ')
		}
	}
	content = result.String()

	// 确保内容是有效的UTF-8
	if !utf8.ValidString(content) {
		// 如果不是有效的UTF-8，替换无效字符
		content = strings.ToValidUTF8(content, "")
	}

	// 移除零宽字符和其他特殊控制字符
	content = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, content)

	// 返回过滤后的内容
	return content
}
