package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// SettingsHandler 系统设置 API 处理器
type SettingsHandler struct {
	DB *sqlx.DB
}

// NewSettingsHandler 创建设置 Handler
func NewSettingsHandler(db *sqlx.DB) *SettingsHandler {
	return &SettingsHandler{DB: db}
}

// StorageConfigResponse 存储配置响应（脱敏）
type StorageConfigResponse struct {
	Driver     string `json:"driver"`
	LocalPath  string `json:"local_path"`
	S3Endpoint string `json:"s3_endpoint"`
	S3Bucket   string `json:"s3_bucket"`
	S3Region   string `json:"s3_region"`
	S3Key      string `json:"s3_key"` // 脱敏显示
	PublicURL  string `json:"public_url"`
}

// StorageConfigRequest 存储配置更新请求
type StorageConfigRequest struct {
	Driver     string `json:"driver" binding:"required,oneof=local s3"`
	LocalPath  string `json:"local_path"`
	S3Endpoint string `json:"s3_endpoint"`
	S3Bucket   string `json:"s3_bucket"`
	S3Region   string `json:"s3_region"`
	S3Key      string `json:"s3_key"`
	S3Secret   string `json:"s3_secret"`
	PublicURL  string `json:"public_url"`
}

// getConfigValue 从 system_configs 表读取单个配置值
func (h *SettingsHandler) getConfigValue(key string) (string, error) {
	var value string
	err := h.DB.Get(&value, `SELECT value FROM system_configs WHERE account_id IS NULL AND key = $1`, key)
	if err != nil {
		return "", err
	}
	return value, nil
}

// upsertConfig 插入或更新配置项
func (h *SettingsHandler) upsertConfig(key, value string) error {
	_, err := h.DB.Exec(`
		INSERT INTO system_configs (account_id, key, value, type)
		VALUES (NULL, $1, $2, 'string')
		ON CONFLICT (COALESCE(account_id, 0), key)
		DO UPDATE SET value = $2, updated_at = NOW()
	`, key, value)
	return err
}

// GetStorageConfig GET /api/v1/admin/settings — 获取当前存储配置
func (h *SettingsHandler) GetStorageConfig(c *gin.Context) {
	keys := []string{
		"storage.driver",
		"storage.local_path",
		"storage.s3_endpoint",
		"storage.s3_bucket",
		"storage.s3_region",
		"storage.s3_key",
		"storage.public_url",
	}

	resp := StorageConfigResponse{}
	for _, key := range keys {
		value, err := h.getConfigValue(key)
		if err != nil {
			// 配置不存在时忽略（使用默认值）
			continue
		}
		switch key {
		case "storage.driver":
			resp.Driver = value
		case "storage.local_path":
			resp.LocalPath = value
		case "storage.s3_endpoint":
			resp.S3Endpoint = value
		case "storage.s3_bucket":
			resp.S3Bucket = value
		case "storage.s3_region":
			resp.S3Region = value
		case "storage.s3_key":
			// 脱敏：只显示前4位和后4位
			if len(value) > 8 {
				resp.S3Key = value[:4] + "****" + value[len(value)-4:]
			} else if value != "" {
				resp.S3Key = "****"
			}
		case "storage.public_url":
			resp.PublicURL = value
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": resp,
	})
}

// UpdateStorageConfig PUT /api/v1/admin/settings — 更新存储配置
func (h *SettingsHandler) UpdateStorageConfig(c *gin.Context) {
	var req StorageConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 40001,
			"msg":  "参数错误: " + err.Error(),
			"data": nil,
		})
		return
	}

	configs := map[string]string{
		"storage.driver":      req.Driver,
		"storage.local_path":  req.LocalPath,
		"storage.s3_endpoint": req.S3Endpoint,
		"storage.s3_bucket":   req.S3Bucket,
		"storage.s3_region":   req.S3Region,
		"storage.s3_key":      req.S3Key,
		"storage.public_url":  req.PublicURL,
	}

	// 只有当 S3Secret 不为空时才更新（避免清空已有密钥）
	if req.S3Secret != "" {
		configs["storage.s3_secret"] = req.S3Secret
	}

	for key, value := range configs {
		if err := h.upsertConfig(key, value); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": 50001,
				"msg":  "保存配置失败: " + key,
				"data": nil,
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "保存成功",
		"data": nil,
	})
}
