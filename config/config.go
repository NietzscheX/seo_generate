package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 配置结构
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	AI       AIConfig       `mapstructure:"ai"`
	Auth     AuthConfig     `mapstructure:"auth"`
	API5118  API5118Config  `mapstructure:"api_5118"`
	Content  ContentConfig  `mapstructure:"content"`
	SEO      SEOConfig      `mapstructure:"seo"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port string `mapstructure:"port"`
	Env  string `mapstructure:"env"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host           string `mapstructure:"host"`
	Port           string `mapstructure:"port"`
	User           string `mapstructure:"user"`
	Password       string `mapstructure:"password"`
	Name           string `mapstructure:"name"`
	MaxConnections int    `mapstructure:"max_connections"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// AIConfig AI配置
type AIConfig struct {
	Model          string  `mapstructure:"model"`
	APIKey         string  `mapstructure:"api_key"`
	Timeout        int     `mapstructure:"timeout"`
	MaxTokens      int     `mapstructure:"max_tokens"`
	Temperature    float64 `mapstructure:"temperature"`
	DeepseekAPIKey string  `mapstructure:"deepseek_api_key"`
	DeepseekAPIURL string  `mapstructure:"deepseek_api_url"`
	OllamaEndpoint string  `mapstructure:"ollama_endpoint"`
}

// AuthConfig 认证配置
type AuthConfig struct {
	JWTSecret          string        `mapstructure:"jwt_secret"`
	AccessTokenExpiry  time.Duration `mapstructure:"access_token_expiry"`
	RefreshTokenExpiry time.Duration `mapstructure:"refresh_token_expiry"`
}

// API5118Config 5118 API配置
type API5118Config struct {
	Key     string `mapstructure:"key"`
	BaseURL string `mapstructure:"base_url"`
}

// ContentConfig 内容生成配置
type ContentConfig struct {
	ArticleMinLength int `mapstructure:"article_min_length"`
	ArticleMaxLength int `mapstructure:"article_max_length"`
}

// SEOConfig SEO配置
type SEOConfig struct {
	SiteURL  string `mapstructure:"site_url"`
	SiteName string `mapstructure:"site_name"`
}

// LoadConfig 从配置文件和环境变量加载配置
func LoadConfig() (*Config, error) {
	fmt.Println("开始加载配置文件...")

	// 设置viper读取.env文件
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")

	// 启用环境变量替换
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 读取.env文件
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	// 手动设置配置键的映射
	viper.Set("database.host", viper.GetString("DB_HOST"))
	viper.Set("database.port", viper.GetString("DB_PORT"))
	viper.Set("database.user", viper.GetString("DB_USER"))
	viper.Set("database.password", viper.GetString("DB_PASSWORD"))
	viper.Set("database.name", viper.GetString("DB_NAME"))
	viper.Set("database.max_connections", viper.GetInt("DB_MAX_CONNECTIONS"))

	viper.Set("server.port", viper.GetString("PORT"))
	viper.Set("server.env", viper.GetString("ENV"))

	viper.Set("redis.host", viper.GetString("REDIS_HOST"))
	viper.Set("redis.port", viper.GetString("REDIS_PORT"))
	viper.Set("redis.password", viper.GetString("REDIS_PASSWORD"))
	viper.Set("redis.db", viper.GetInt("REDIS_DB"))

	viper.Set("ai.model", viper.GetString("AI_MODEL"))
	viper.Set("ai.api_key", viper.GetString("AI_API_KEY"))
	viper.Set("ai.timeout", viper.GetInt("AI_TIMEOUT"))
	viper.Set("ai.max_tokens", viper.GetInt("AI_MAX_TOKENS"))
	viper.Set("ai.temperature", viper.GetFloat64("AI_TEMPERATURE"))
	viper.Set("ai.deepseek_api_key", viper.GetString("AI_DEEPSEEK_API_KEY"))
	viper.Set("ai.deepseek_api_url", viper.GetString("AI_DEEPSEEK_API_URL"))
	viper.Set("ai.ollama_endpoint", viper.GetString("AI_OLLAMA_ENDPOINT"))

	viper.Set("api_5118.key", viper.GetString("API_5118_KEY"))
	viper.Set("api_5118.base_url", viper.GetString("API_5118_BASE_URL"))

	viper.Set("content.article_min_length", viper.GetInt("ARTICLE_MIN_LENGTH"))
	viper.Set("content.article_max_length", viper.GetInt("ARTICLE_MAX_LENGTH"))

	viper.Set("seo.site_url", viper.GetString("SITE_URL"))
	viper.Set("seo.site_name", viper.GetString("SITE_NAME"))

	viper.Set("auth.jwt_secret", viper.GetString("JWT_SECRET"))
	viper.Set("auth.access_token_expiry", viper.GetDuration("ACCESS_TOKEN_EXPIRY"))
	viper.Set("auth.refresh_token_expiry", viper.GetDuration("REFRESH_TOKEN_EXPIRY"))

	fmt.Printf("环境变量数据库配置: host=%s port=%s user=%s password=%s dbname=%s\n",
		viper.GetString("DB_HOST"),
		viper.GetString("DB_PORT"),
		viper.GetString("DB_USER"),
		viper.GetString("DB_PASSWORD"),
		viper.GetString("DB_NAME"),
	)

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析配置失败: %v", err)
	}

	fmt.Printf("解析后的数据库配置: host=%s port=%s user=%s password=%s dbname=%s\n",
		config.Database.Host,
		config.Database.Port,
		config.Database.User,
		config.Database.Password,
		config.Database.Name,
	)

	return &config, nil
}
