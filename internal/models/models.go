package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Category 分类模型
type Category struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"size:100;not null;uniqueIndex" json:"name"`
	ParentID  *uint          `gorm:"default:null" json:"parent_id"`
	Parent    *Category      `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children  []Category     `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Keywords  []Keyword      `gorm:"many2many:category_keywords;" json:"keywords,omitempty"`
	Articles  []Article      `gorm:"many2many:category_articles;" json:"articles,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Keyword 关键词模型
type Keyword struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Word         string         `gorm:"size:200;not null;uniqueIndex" json:"word"`
	SearchVolume int            `gorm:"default:0" json:"search_volume"`
	Categories   []Category     `gorm:"many2many:category_keywords;" json:"categories,omitempty"`
	Articles     []Article      `gorm:"many2many:keyword_articles;" json:"articles,omitempty"`
	Source       string         `gorm:"size:50;default:'5118'" json:"source"`
	Status       string         `gorm:"size:20;default:'active'" json:"status"` // active, inactive, pending
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// Article 文章模型
type Article struct {
	ID          uint       `json:"id" gorm:"primarykey"`
	Title       string     `json:"title" gorm:"not null"`
	Slug        string     `json:"slug" gorm:"uniqueIndex"`
	Content     string     `json:"content" gorm:"type:text"`
	Summary     string     `json:"summary"`
	MetaTitle   string     `json:"meta_title"`
	MetaDesc    string     `json:"meta_desc"`
	Status      string     `json:"status" gorm:"default:draft"`
	ViewCount   int        `json:"view_count" gorm:"default:0"`
	PublishedAt *time.Time `json:"published_at"`
	UserID      *uint      `json:"user_id"`
	User        *User      `json:"user,omitempty"`
	Categories  []Category `json:"categories" gorm:"many2many:article_categories;"`
	Keywords    []Keyword  `json:"keywords" gorm:"many2many:article_keywords;"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty" gorm:"index"`
}

// GenerationTask 内容生成任务模型
type GenerationTask struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	KeywordID    uint           `json:"keyword_id"`
	Keyword      Keyword        `gorm:"foreignKey:KeywordID" json:"keyword"`
	Status       string         `gorm:"size:20;default:'pending'" json:"status"` // pending, processing, completed, failed
	ArticleID    *uint          `json:"article_id"`
	Article      *Article       `gorm:"foreignKey:ArticleID" json:"article,omitempty"`
	Prompt       string         `gorm:"type:text" json:"prompt"`
	ErrorMessage string         `gorm:"type:text" json:"error_message"`
	ModelUsed    string         `gorm:"size:50" json:"model_used"` // deepseek, ollama
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// APILog API调用日志模型
type APILog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	APIName   string    `gorm:"size:50;not null" json:"api_name"` // 5118, deepseek, ollama
	Endpoint  string    `gorm:"size:200;not null" json:"endpoint"`
	Request   string    `gorm:"type:text" json:"request"`
	Response  string    `gorm:"type:text" json:"response"`
	Status    int       `json:"status"`
	Duration  int       `json:"duration"` // 毫秒
	CreatedAt time.Time `json:"created_at"`
}

// User 用户模型
type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Username  string         `gorm:"size:50;not null;uniqueIndex" json:"username"`
	Email     string         `gorm:"size:100;not null;uniqueIndex" json:"email"`
	Password  string         `gorm:"size:100;not null" json:"-"`         // 不在JSON中返回密码
	Role      string         `gorm:"size:20;default:'user'" json:"role"` // admin, editor, user
	Active    bool           `gorm:"default:true" json:"active"`
	LastLogin *time.Time     `json:"last_login"`
	Tokens    []Token        `json:"-"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Token 认证令牌模型
type Token struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `json:"user_id"`
	User      User           `gorm:"foreignKey:UserID" json:"-"`
	Token     string         `gorm:"size:100;not null;uniqueIndex" json:"token"`
	Type      string         `gorm:"size:20;default:'access'" json:"type"` // access, refresh
	ExpiresAt time.Time      `json:"expires_at"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// SetPassword 设置用户密码（加密）
func (u *User) SetPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

// CheckPassword 检查密码是否正确
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// AutoMigrate 自动迁移数据库表结构
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Category{},
		&Keyword{},
		&Article{},
		&GenerationTask{},
		&APILog{},
		&User{},
		&Token{},
	)
}
