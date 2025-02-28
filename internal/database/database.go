package database

import (
	"fmt"
	"time"

	"github.com/NietzscheX/seo-generate/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewDatabase 创建数据库连接
func NewDatabase(cfg *config.Config) (*gorm.DB, error) {
	fmt.Printf("数据库配置: host=%s port=%s user=%s password=%s dbname=%s\n",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
	)

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=Asia/Shanghai client_encoding=UTF8",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
	)

	// 配置GORM日志
	logLevel := logger.Silent
	if cfg.Server.Env == "development" {
		logLevel = logger.Info
	}

	// 打开数据库连接
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("无法连接到数据库: %w", err)
	}

	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("无法获取数据库连接: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(cfg.Database.MaxConnections)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}
