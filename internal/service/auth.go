// Package service 认证业务逻辑层
// 密码哈希、Token 生成、登录逻辑
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	"github.com/weiyeston/weiyeston-v2/internal/config"
	"github.com/weiyeston/weiyeston-v2/internal/model"
)

// HashPassword 使用 bcrypt (cost=12) 生成密码哈希
func HashPassword(password string) (string, error) {
	if len(password) > 72 {
		return "", fmt.Errorf("密码长度不能超过 72 字节")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", fmt.Errorf("密码哈希生成失败: %w", err)
	}
	return string(hash), nil
}

// VerifyPassword 验证密码与哈希是否匹配
func VerifyPassword(hash, password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return fmt.Errorf("密码验证失败: %w", err)
	}
	return nil
}

// GenerateAccessToken 生成 Access Token (JWT HS256)
// 使用 jwt.MapClaims，sub 为 int64
func GenerateAccessToken(user *model.Tenant, cfg config.JWTConfig) (string, error) {
	now := time.Now()
	expiration := cfg.GetAccessExpiration()
	if expiration <= 0 {
		expiration = 2 * time.Hour
	}

	claims := jwt.MapClaims{
		"sub":       user.ID,
		"tenant_id": user.ID,
		"role":      user.Role,
		"nickname":  user.Nickname,
		"iat":       now.Unix(),
		"exp":       now.Add(expiration).Unix(),
		"iss":       cfg.Issuer,
		"jti":       uuid.New().String(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.Secret))
	if err != nil {
		return "", fmt.Errorf("生成 Access Token 失败: %w", err)
	}
	return tokenString, nil
}

// GenerateRefreshToken 生成 Refresh Token (UUID v4)，存入 Redis
// Key 格式: refresh_token:{user_id}
func GenerateRefreshToken(ctx context.Context, rdb *redis.Client, userID int64, cfg config.JWTConfig) (string, error) {
	token := uuid.New().String()
	key := fmt.Sprintf("refresh_token:%d", userID)
	ttl := cfg.GetRefreshExpiration()
	if ttl <= 0 {
		ttl = 7 * 24 * time.Hour
	}
	if err := rdb.Set(ctx, key, token, ttl).Err(); err != nil {
		return "", fmt.Errorf("存储 Refresh Token 失败: %w", err)
	}
	return token, nil
}

// RevokeRefreshToken 撤销用户的 Refresh Token（登出时调用）
func RevokeRefreshToken(ctx context.Context, rdb *redis.Client, userID int64) error {
	key := fmt.Sprintf("refresh_token:%d", userID)
	return rdb.Del(ctx, key).Err()
}

// GetRefreshToken 从 Redis 获取用户的 Refresh Token
func GetRefreshToken(ctx context.Context, rdb *redis.Client, userID int64) (string, error) {
	key := fmt.Sprintf("refresh_token:%d", userID)
	val, err := rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}
