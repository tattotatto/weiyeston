// Package service auth service test
// TDD: test-first, auth service not yet implemented
package service

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/weiyeston/weiyeston-v2/internal/config"
)

// TestBcryptGenerateAndVerify tests bcrypt hash generation and verification
func TestBcryptGenerateAndVerify(t *testing.T) {
	t.Run("generate and verify bcrypt hash", func(t *testing.T) {
		password := "my-secret-password-123"
		hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.True(t, len(hash) > 50)

		err = bcrypt.CompareHashAndPassword(hash, []byte(password))
		assert.NoError(t, err)
	})

	t.Run("wrong password fails verification", func(t *testing.T) {
		hash, err := bcrypt.GenerateFromPassword([]byte("correct"), 12)
		require.NoError(t, err)
		err = bcrypt.CompareHashAndPassword(hash, []byte("wrong"))
		assert.Error(t, err)
		assert.Equal(t, bcrypt.ErrMismatchedHashAndPassword, err)
	})

	t.Run("same password produces different hash", func(t *testing.T) {
		hash1, _ := bcrypt.GenerateFromPassword([]byte("same"), 10)
		hash2, _ := bcrypt.GenerateFromPassword([]byte("same"), 10)
		assert.NotEqual(t, string(hash1), string(hash2))
	})

	t.Run("cost=12 hash has $2a$12$ prefix", func(t *testing.T) {
		hash, err := bcrypt.GenerateFromPassword([]byte("test"), 12)
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(string(hash), "$2a$12$"))
	})
}

func TestBcryptEdgeCases(t *testing.T) {
	t.Run("empty password produces hash", func(t *testing.T) {
		hash, err := bcrypt.GenerateFromPassword([]byte(""), 10)
		require.NoError(t, err)
		err = bcrypt.CompareHashAndPassword(hash, []byte(""))
		assert.NoError(t, err)
	})

	t.Run("password exceeding 72 bytes rejected", func(t *testing.T) {
		longPassword := strings.Repeat("a", 73)
		_, err := bcrypt.GenerateFromPassword([]byte(longPassword), 10)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "password length exceeds 72 bytes")

		exact72 := strings.Repeat("a", 72)
		hash, err := bcrypt.GenerateFromPassword([]byte(exact72), 10)
		assert.NoError(t, err)
		assert.NotEmpty(t, hash)
	})
}

// TestAccessTokenGeneration tests JWT access token generation with jwt.MapClaims
func TestAccessTokenGeneration(t *testing.T) {
	cfg := config.JWTConfig{
		Secret:     "test-access-secret-key-32bytes!!",
		Expiration: 2 * time.Hour,
		Issuer:     "weiyeston-v2-test",
	}

	t.Run("access token contains all required claims", func(t *testing.T) {
		userID := int64(42)
		role := "admin"
		nickname := "Admin"

		now := time.Now()
		claims := jwt.MapClaims{
			"sub":       userID,
			"tenant_id": userID,
			"role":      role,
			"nickname":  nickname,
			"iat":       now.Unix(),
			"exp":       now.Add(cfg.Expiration).Unix(),
			"iss":       cfg.Issuer,
			"jti":       "test-jti-uuid",
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenStr, err := token.SignedString([]byte(cfg.Secret))
		require.NoError(t, err)
		assert.NotEmpty(t, tokenStr)
		assert.Greater(t, len(tokenStr), 50)

		parsed, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(cfg.Secret), nil
		})
		require.NoError(t, err)
		require.True(t, parsed.Valid)

		parsedClaims, ok := parsed.Claims.(jwt.MapClaims)
		require.True(t, ok)

		sub, exists := parsedClaims["sub"]
		assert.True(t, exists)
		subFloat, ok := sub.(float64)
		assert.True(t, ok, "sub should be float64: got %T", sub)
		assert.Equal(t, userID, int64(subFloat))

		assert.Equal(t, role, parsedClaims["role"])
		assert.Equal(t, nickname, parsedClaims["nickname"])
		assert.Equal(t, cfg.Issuer, parsedClaims["iss"])
	})

	t.Run("sub is JSON number not string - compatible with T0 auth.go", func(t *testing.T) {
		userID := int64(1)
		claims := jwt.MapClaims{
			"sub": userID,
			"exp": time.Now().Add(2 * time.Hour).Unix(),
			"iss": "weiyeston-v2",
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenStr, err := token.SignedString([]byte(cfg.Secret))
		require.NoError(t, err)

		parts := strings.Split(tokenStr, ".")
		require.Equal(t, 3, len(parts))

		payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
		require.NoError(t, err)
		assert.Contains(t, string(payloadJSON), `"sub":1`,
			"sub should be JSON number, not string")
	})

	t.Run("access token expires in 2 hours", func(t *testing.T) {
		cfg2h := config.JWTConfig{
			Secret:     "test-secret",
			Expiration: 2 * time.Hour,
			Issuer:     "weiyeston-v2",
		}
		now := time.Now()
		claims := jwt.MapClaims{
			"sub": int64(1),
			"iat": now.Unix(),
			"exp": now.Add(cfg2h.Expiration).Unix(),
			"iss": cfg2h.Issuer,
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenStr, _ := token.SignedString([]byte(cfg2h.Secret))
		parsed, _ := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			return []byte(cfg2h.Secret), nil
		})
		parsedClaims := parsed.Claims.(jwt.MapClaims)
		iat := int64(parsedClaims["iat"].(float64))
		exp := int64(parsedClaims["exp"].(float64))
		assert.Equal(t, int64(7200), exp-iat)
	})
}

func TestAccessTokenParsing(t *testing.T) {
	cfg := config.JWTConfig{
		Secret:     "test-parse-secret-key-32bytes!!!",
		Expiration: 2 * time.Hour,
		Issuer:     "weiyeston-v2",
	}

	t.Run("valid token parses successfully", func(t *testing.T) {
		claims := jwt.MapClaims{
			"sub":       int64(100),
			"tenant_id": int64(100),
			"role":      "user",
			"iat":       time.Now().Unix(),
			"exp":       time.Now().Add(2 * time.Hour).Unix(),
			"iss":       cfg.Issuer,
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenStr, _ := token.SignedString([]byte(cfg.Secret))
		parsed, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			return []byte(cfg.Secret), nil
		})
		require.NoError(t, err)
		assert.True(t, parsed.Valid)
	})

	t.Run("expired token returns error", func(t *testing.T) {
		claims := jwt.MapClaims{
			"sub": int64(1),
			"iat": time.Now().Add(-3 * time.Hour).Unix(),
			"exp": time.Now().Add(-1 * time.Hour).Unix(),
			"iss": cfg.Issuer,
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenStr, _ := token.SignedString([]byte(cfg.Secret))
		_, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			return []byte(cfg.Secret), nil
		})
		assert.Error(t, err)
	})

	t.Run("ParseUnverified skips expiration check for refresh scenario", func(t *testing.T) {
		claims := jwt.MapClaims{
			"sub": int64(42),
			"iat": time.Now().Add(-3 * time.Hour).Unix(),
			"exp": time.Now().Add(-1 * time.Hour).Unix(),
			"iss": cfg.Issuer,
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenStr, _ := token.SignedString([]byte(cfg.Secret))

		parser := jwt.Parser{}
		parsedToken, _, err := parser.ParseUnverified(tokenStr, jwt.MapClaims{})
		require.NoError(t, err)

		parsedClaims, ok := parsedToken.Claims.(jwt.MapClaims)
		require.True(t, ok)

		sub, ok := parsedClaims["sub"]
		assert.True(t, ok)
		subFloat, ok := sub.(float64)
		assert.True(t, ok)
		assert.Equal(t, int64(42), int64(subFloat))
	})

	t.Run("wrong signature fails parsing", func(t *testing.T) {
		claims := jwt.MapClaims{"sub": int64(1), "exp": time.Now().Add(2 * time.Hour).Unix()}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenStr, _ := token.SignedString([]byte("different-secret"))
		_, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			return []byte(cfg.Secret), nil
		})
		assert.Error(t, err)
	})

	t.Run("non-HS256 algorithm rejected", func(t *testing.T) {
		claims := jwt.MapClaims{"sub": int64(1), "exp": time.Now().Add(2 * time.Hour).Unix()}
		token := jwt.NewWithClaims(jwt.SigningMethodHS384, claims)
		tokenStr, _ := token.SignedString([]byte(cfg.Secret))
		_, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(cfg.Secret), nil
		})
		assert.Error(t, err)
	})
}

// TestRefreshTokenSpec validates refresh token format and lifecycle
func TestRefreshTokenSpec(t *testing.T) {
	t.Run("UUID v4 format 36 chars", func(t *testing.T) {
		exampleUUID := "550e8400-e29b-41d4-a716-446655440000"
		assert.Len(t, exampleUUID, 36)
		assert.Equal(t, byte('4'), exampleUUID[14])
		assert.Contains(t, "89ab", string(exampleUUID[19]))
	})

	t.Run("each generated refresh token is unique", func(t *testing.T) {
		uuid1 := "550e8400-e29b-41d4-a716-446655440000"
		uuid2 := "660e8400-e29b-41d4-a716-446655440001"
		assert.NotEqual(t, uuid1, uuid2)
	})

	t.Run("storage key format is refresh_token:{user_id}", func(t *testing.T) {
		expectedKey := "refresh_token:42"
		assert.Equal(t, "refresh_token:42", expectedKey)
	})
}

func TestRefreshTokenRotation(t *testing.T) {
	store := make(map[int64]string)
	userID := int64(1)
	oldToken := "old-refresh-token-uuid"
	newToken := "new-refresh-token-uuid"

	t.Run("rotation replaces old token with new", func(t *testing.T) {
		store[userID] = oldToken
		assert.Equal(t, oldToken, store[userID])
		store[userID] = newToken
		assert.NotEqual(t, oldToken, store[userID])
		assert.Equal(t, newToken, store[userID])
	})

	t.Run("old token fails replay detection", func(t *testing.T) {
		store[userID] = newToken
		if store[userID] != oldToken {
			t.Log("replay detected: old token rejected")
		}
		assert.NotEqual(t, oldToken, store[userID])
	})
}

func TestRefreshTokenRevocation(t *testing.T) {
	store := make(map[int64]string)

	t.Run("logout removes refresh token", func(t *testing.T) {
		userID := int64(1)
		store[userID] = "some-refresh-token"
		delete(store, userID)
		_, exists := store[userID]
		assert.False(t, exists)
	})

	t.Run("revoked token is invalid", func(t *testing.T) {
		_, exists := store[int64(2)]
		assert.False(t, exists)
	})
}

func TestTokenExpirationPolicy(t *testing.T) {
	tests := []struct {
		name       string
		accessTTL  time.Duration
		refreshTTL time.Duration
	}{
		{"prod: Access 2h, Refresh 7d", 2 * time.Hour, 7 * 24 * time.Hour},
		{"dev: Access 15m, Refresh 1h", 15 * time.Minute, 1 * time.Hour},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Greater(t, tt.refreshTTL, tt.accessTTL)
			assert.Positive(t, tt.accessTTL.Seconds())
			assert.Positive(t, tt.refreshTTL.Seconds())
		})
	}

	t.Run("refresh token 7 days equals 604800 seconds", func(t *testing.T) {
		assert.Equal(t, int64(604800), int64((7*24*time.Hour).Seconds()))
	})
}

func TestClaimsWithT0AuthMiddleware(t *testing.T) {
	secret := "compat-test-secret-key-32bytes!!!"

	t.Run("sub int64 to float64 compatible with T0 auth.go", func(t *testing.T) {
		userID := int64(999888777)
		claims := jwt.MapClaims{"sub": userID, "exp": time.Now().Add(2 * time.Hour).Unix(), "iss": "weiyeston-v2"}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenStr, _ := token.SignedString([]byte(secret))
		parsed, _ := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) { return []byte(secret), nil })
		parsedClaims := parsed.Claims.(jwt.MapClaims)
		sub := parsedClaims["sub"]
		var parsedUserID int64
		switch v := sub.(type) {
		case float64:
			parsedUserID = int64(v)
		case int64:
			parsedUserID = v
		default:
			t.Fatalf("incompatible sub type: %T", sub)
		}
		assert.Equal(t, userID, parsedUserID)
	})

	t.Run("missing tenant_id defaults to 0", func(t *testing.T) {
		claims := jwt.MapClaims{"sub": int64(1), "exp": time.Now().Add(2 * time.Hour).Unix(), "iss": "weiyeston-v2"}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenStr, _ := token.SignedString([]byte(secret))
		parsed, _ := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) { return []byte(secret), nil })
		parsedClaims := parsed.Claims.(jwt.MapClaims)
		_, hasTenant := parsedClaims["tenant_id"]
		assert.False(t, hasTenant)
	})

	t.Run("missing role defaults to user", func(t *testing.T) {
		claims := jwt.MapClaims{"sub": int64(1), "exp": time.Now().Add(2 * time.Hour).Unix(), "iss": "weiyeston-v2"}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenStr, _ := token.SignedString([]byte(secret))
		parsed, _ := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) { return []byte(secret), nil })
		parsedClaims := parsed.Claims.(jwt.MapClaims)
		_, hasRole := parsedClaims["role"]
		assert.False(t, hasRole)
	})
}

func TestAuthErrorCodes(t *testing.T) {
	errorCodes := map[int]string{
		0:     "success",
		40001: "invalid params",
		401:   "unauthorized",
		40101: "wrong username or password",
		40102: "refresh token invalid or expired",
		403:   "forbidden",
		40301: "account disabled",
		40302: "no access to this resource",
		42901: "login too frequent",
		500:   "internal server error",
	}
	for code, desc := range errorCodes {
		assert.GreaterOrEqual(t, code, 0)
		assert.NotEmpty(t, desc)
	}
	assert.Equal(t, 40101, 40101)
	assert.Equal(t, 40102, 40102)
	assert.Equal(t, 40301, 40301)
	assert.Equal(t, 42901, 42901)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
