package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/NietzscheX/seo-generate/config"
	"github.com/NietzscheX/seo-generate/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

// AuthService 认证服务
type AuthService struct {
	db     *gorm.DB
	config *config.Config
}

// NewAuthService 创建认证服务
func NewAuthService(db *gorm.DB, cfg *config.Config) *AuthService {
	return &AuthService{
		db:     db,
		config: cfg,
	}
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// TokenResponse 令牌响应
type TokenResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
	UserID       uint      `json:"user_id"`
	Username     string    `json:"username"`
	Role         string    `json:"role"`
}

// Register 用户注册
func (s *AuthService) Register(req RegisterRequest) (*models.User, error) {
	// 检查用户名是否已存在
	var existingUser models.User
	if err := s.db.Where("username = ?", req.Username).First(&existingUser).Error; err == nil {
		return nil, errors.New("用户名已存在")
	} else if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	// 检查邮箱是否已存在
	if err := s.db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		return nil, errors.New("邮箱已存在")
	} else if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	// 创建新用户
	user := models.User{
		Username: req.Username,
		Email:    req.Email,
		Role:     "user", // 默认角色
		Active:   true,
	}

	// 设置密码
	if err := user.SetPassword(req.Password); err != nil {
		return nil, fmt.Errorf("设置密码失败: %w", err)
	}

	// 保存用户
	if err := s.db.Create(&user).Error; err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	return &user, nil
}

// Login 用户登录
func (s *AuthService) Login(req LoginRequest) (*TokenResponse, error) {
	// 查找用户
	var user models.User
	if err := s.db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("用户名或密码错误")
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	// 检查用户是否激活
	if !user.Active {
		return nil, errors.New("用户已被禁用")
	}

	// 验证密码
	if !user.CheckPassword(req.Password) {
		return nil, errors.New("用户名或密码错误")
	}

	// 生成令牌
	accessToken, refreshToken, expiresAt, err := s.generateTokens(user)
	if err != nil {
		return nil, fmt.Errorf("生成令牌失败: %w", err)
	}

	// 更新最后登录时间
	now := time.Now()
	s.db.Model(&user).Update("last_login", &now)

	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		TokenType:    "Bearer",
		UserID:       user.ID,
		Username:     user.Username,
		Role:         user.Role,
	}, nil
}

// RefreshToken 刷新令牌
func (s *AuthService) RefreshToken(refreshToken string) (*TokenResponse, error) {
	// 验证刷新令牌
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("非预期的签名方法: %v", token.Header["alg"])
		}
		return []byte(s.config.Auth.JWTSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("解析令牌失败: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("无效的令牌")
	}

	// 获取令牌中的用户ID
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("无效的令牌声明")
	}

	userID, ok := claims["user_id"].(float64)
	if !ok {
		return nil, errors.New("无效的用户ID")
	}

	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "refresh" {
		return nil, errors.New("无效的令牌类型")
	}

	// 查找用户
	var user models.User
	if err := s.db.First(&user, uint(userID)).Error; err != nil {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	// 检查用户是否激活
	if !user.Active {
		return nil, errors.New("用户已被禁用")
	}

	// 生成新令牌
	accessToken, newRefreshToken, expiresAt, err := s.generateTokens(user)
	if err != nil {
		return nil, fmt.Errorf("生成令牌失败: %w", err)
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    expiresAt,
		TokenType:    "Bearer",
		UserID:       user.ID,
		Username:     user.Username,
		Role:         user.Role,
	}, nil
}

// generateTokens 生成访问令牌和刷新令牌
func (s *AuthService) generateTokens(user models.User) (string, string, time.Time, error) {
	// 设置过期时间
	accessExpiresAt := time.Now().Add(time.Hour * 24)      // 访问令牌24小时过期
	refreshExpiresAt := time.Now().Add(time.Hour * 24 * 7) // 刷新令牌7天过期

	// 创建访问令牌
	accessClaims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"role":     user.Role,
		"type":     "access",
		"exp":      accessExpiresAt.Unix(),
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(s.config.Auth.JWTSecret))
	if err != nil {
		return "", "", time.Time{}, err
	}

	// 创建刷新令牌
	refreshClaims := jwt.MapClaims{
		"user_id": user.ID,
		"type":    "refresh",
		"exp":     refreshExpiresAt.Unix(),
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(s.config.Auth.JWTSecret))
	if err != nil {
		return "", "", time.Time{}, err
	}

	// 保存令牌到数据库
	dbToken := models.Token{
		UserID:    user.ID,
		Token:     accessTokenString,
		Type:      "access",
		ExpiresAt: accessExpiresAt,
	}
	if err := s.db.Create(&dbToken).Error; err != nil {
		return "", "", time.Time{}, err
	}

	dbRefreshToken := models.Token{
		UserID:    user.ID,
		Token:     refreshTokenString,
		Type:      "refresh",
		ExpiresAt: refreshExpiresAt,
	}
	if err := s.db.Create(&dbRefreshToken).Error; err != nil {
		return "", "", time.Time{}, err
	}

	return accessTokenString, refreshTokenString, accessExpiresAt, nil
}

// GetUserFromToken 从令牌获取用户
func (s *AuthService) GetUserFromToken(tokenString string) (*models.User, error) {
	// 解析令牌
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("非预期的签名方法: %v", token.Header["alg"])
		}
		return []byte(s.config.Auth.JWTSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("解析令牌失败: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("无效的令牌")
	}

	// 获取令牌中的用户ID
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("无效的令牌声明")
	}

	userID, ok := claims["user_id"].(float64)
	if !ok {
		return nil, errors.New("无效的用户ID")
	}

	// 查找用户
	var user models.User
	if err := s.db.First(&user, uint(userID)).Error; err != nil {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	return &user, nil
}

// AuthMiddleware 认证中间件
func (s *AuthService) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取令牌
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"code": 401, "message": "未提供认证令牌"})
			c.Abort()
			return
		}

		// 检查令牌格式
		if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
			c.JSON(401, gin.H{"code": 401, "message": "认证令牌格式错误"})
			c.Abort()
			return
		}

		tokenString := authHeader[7:]

		// 获取用户
		user, err := s.GetUserFromToken(tokenString)
		if err != nil {
			c.JSON(401, gin.H{"code": 401, "message": "无效的认证令牌"})
			c.Abort()
			return
		}

		// 检查用户是否激活
		if !user.Active {
			c.JSON(403, gin.H{"code": 403, "message": "用户已被禁用"})
			c.Abort()
			return
		}

		// 将用户信息存储到上下文中
		c.Set("user", user)
		c.Next()
	}
}

// RoleMiddleware 角色中间件
func (s *AuthService) RoleMiddleware(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户
		userInterface, exists := c.Get("user")
		if !exists {
			c.JSON(401, gin.H{"code": 401, "message": "未认证"})
			c.Abort()
			return
		}

		user, ok := userInterface.(*models.User)
		if !ok {
			c.JSON(500, gin.H{"code": 500, "message": "服务器内部错误"})
			c.Abort()
			return
		}

		// 检查用户角色
		hasRole := false
		for _, role := range roles {
			if user.Role == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			c.JSON(403, gin.H{"code": 403, "message": "权限不足"})
			c.Abort()
			return
		}

		c.Next()
	}
}
