package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/NietzscheX/seo-generate/config"
	"github.com/NietzscheX/seo-generate/internal/models"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	ArticleQueueKey = "article:queue"
	ArticleSetKey   = "article:set"
)

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
)

// GenerationTask 生成任务
type GenerationTask struct {
	ID          string     `json:"id"`
	KeywordID   uint       `json:"keyword_id"`
	CategoryIDs []uint     `json:"category_ids"`
	Status      TaskStatus `json:"status"`
	Error       string     `json:"error,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	UserID      uint       `json:"user_id"`
}

// QueueService 队列服务
type QueueService struct {
	db             *gorm.DB
	redis          *redis.Client
	config         *config.Config
	contentService *ContentService
}

// NewQueueService 创建队列服务
func NewQueueService(db *gorm.DB, redis *redis.Client, cfg *config.Config, contentService *ContentService) *QueueService {
	return &QueueService{
		db:             db,
		redis:          redis,
		config:         cfg,
		contentService: contentService,
	}
}

// AddTask 添加任务到队列
func (s *QueueService) AddTask(ctx context.Context, task *GenerationTask) error {
	// 设置任务状态和时间
	task.Status = TaskStatusPending
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()

	// 序列化任务
	taskJSON, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("序列化任务失败: %v", err)
	}

	// 添加到Redis队列和集合
	pipe := s.redis.Pipeline()
	pipe.RPush(ctx, ArticleQueueKey, taskJSON)
	pipe.SAdd(ctx, ArticleSetKey, task.ID)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("添加任务到队列失败: %v", err)
	}

	return nil
}

// GetTask 获取任务信息
func (s *QueueService) GetTask(ctx context.Context, taskID string) (*GenerationTask, error) {
	// 检查任务是否存在
	exists, err := s.redis.SIsMember(ctx, ArticleSetKey, taskID).Result()
	if err != nil {
		return nil, fmt.Errorf("检查任务是否存在失败: %v", err)
	}
	if !exists {
		return nil, fmt.Errorf("任务不存在")
	}

	// 获取任务信息
	taskJSON, err := s.redis.Get(ctx, fmt.Sprintf("task:%s", taskID)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("任务不存在")
		}
		return nil, fmt.Errorf("获取任务信息失败: %v", err)
	}

	var task GenerationTask
	if err := json.Unmarshal([]byte(taskJSON), &task); err != nil {
		return nil, fmt.Errorf("解析任务信息失败: %v", err)
	}

	return &task, nil
}

// ProcessTasks 处理队列中的任务
func (s *QueueService) ProcessTasks(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// 从队列中获取任务
			result, err := s.redis.BLPop(ctx, 0, ArticleQueueKey).Result()
			if err != nil {
				if err != redis.Nil {
					fmt.Printf("获取任务失败: %v\n", err)
				}
				continue
			}

			var task GenerationTask
			if err := json.Unmarshal([]byte(result[1]), &task); err != nil {
				fmt.Printf("解析任务失败: %v\n", err)
				continue
			}

			// 更新任务状态
			task.Status = TaskStatusRunning
			task.UpdatedAt = time.Now()
			s.updateTaskStatus(ctx, &task)

			// 获取关键词
			var keyword models.Keyword
			if err := s.db.First(&keyword, task.KeywordID).Error; err != nil {
				task.Status = TaskStatusFailed
				task.Error = fmt.Sprintf("获取关键词失败: %v", err)
				s.updateTaskStatus(ctx, &task)
				continue
			}

			// 生成文章
			article, err := s.contentService.GenerateArticle(ctx, keyword, task.CategoryIDs)
			if err != nil {
				task.Status = TaskStatusFailed
				task.Error = fmt.Sprintf("生成文章失败: %v", err)
				s.updateTaskStatus(ctx, &task)
				continue
			}

			// 更新任务状态为完成
			task.Status = TaskStatusCompleted
			task.UpdatedAt = time.Now()
			s.updateTaskStatus(ctx, &task)

			// 保存文章作者
			if task.UserID > 0 {
				article.UserID = &task.UserID
				s.db.Save(article)
			}
		}
	}
}

// updateTaskStatus 更新任务状态
func (s *QueueService) updateTaskStatus(ctx context.Context, task *GenerationTask) {
	taskJSON, _ := json.Marshal(task)
	s.redis.Set(ctx, fmt.Sprintf("task:%s", task.ID), taskJSON, 24*time.Hour)
}

// GetTaskList 获取任务列表
func (s *QueueService) GetTaskList(ctx context.Context, userID uint) ([]*GenerationTask, error) {
	// 获取所有任务ID
	taskIDs, err := s.redis.SMembers(ctx, ArticleSetKey).Result()
	if err != nil {
		return nil, fmt.Errorf("获取任务列表失败: %v", err)
	}

	var tasks []*GenerationTask
	for _, taskID := range taskIDs {
		task, err := s.GetTask(ctx, taskID)
		if err != nil {
			continue
		}

		// 只返回用户自己的任务
		if task.UserID == userID {
			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}

// BatchAddTasks 批量添加任务
func (s *QueueService) BatchAddTasks(ctx context.Context, keywordIDs []uint, categoryIDs []uint, userID uint) ([]string, error) {
	var taskIDs []string
	for _, keywordID := range keywordIDs {
		task := &GenerationTask{
			ID:          fmt.Sprintf("task_%d_%d", keywordID, time.Now().UnixNano()),
			KeywordID:   keywordID,
			CategoryIDs: categoryIDs,
			UserID:      userID,
		}

		if err := s.AddTask(ctx, task); err != nil {
			return nil, fmt.Errorf("添加任务失败: %v", err)
		}

		taskIDs = append(taskIDs, task.ID)
	}

	return taskIDs, nil
}
