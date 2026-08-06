package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSettingsTestRouter(handler *SettingsHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	adminGroup := r.Group("/api/v1/admin")
	adminGroup.GET("/settings", handler.GetStorageConfig)
	adminGroup.PUT("/settings", handler.UpdateStorageConfig)
	return r
}

func TestGetStorageConfig_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	// Mock: 每个配置查询返回对应值（顺序必须与 handler 中 keys 一致）
	keys := []string{
		"storage.driver", "storage.local_path", "storage.s3_endpoint",
		"storage.s3_bucket", "storage.s3_region", "storage.s3_key", "storage.public_url",
	}
	values := []string{"local", "./uploads", "", "", "", "", ""}
	for i, key := range keys {
		rows := sqlmock.NewRows([]string{"value"}).AddRow(values[i])
		mock.ExpectQuery(`SELECT value FROM system_configs`).WithArgs(key).WillReturnRows(rows)
	}

	handler := NewSettingsHandler(sqlxDB)
	r := setupSettingsTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/settings", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "local", data["driver"])
	assert.Equal(t, "./uploads", data["local_path"])
}

func TestGetStorageConfig_S3KeyDesensitization(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	keys := []string{
		"storage.driver", "storage.local_path", "storage.s3_endpoint",
		"storage.s3_bucket", "storage.s3_region", "storage.s3_key", "storage.public_url",
	}
	values := []string{
		"s3", "", "https://oss-cn-hangzhou.aliyuncs.com",
		"my-bucket", "cn-hangzhou", "AKID1234567890ABCDEF", "https://cdn.example.com",
	}
	for i, key := range keys {
		rows := sqlmock.NewRows([]string{"value"}).AddRow(values[i])
		mock.ExpectQuery(`SELECT value FROM system_configs`).WithArgs(key).WillReturnRows(rows)
	}

	handler := NewSettingsHandler(sqlxDB)
	r := setupSettingsTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/settings", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})

	// S3 key 应该被脱敏
	s3Key := data["s3_key"].(string)
	assert.Contains(t, s3Key, "****")
	assert.NotEqual(t, "AKID1234567890ABCDEF", s3Key)
}

func TestGetStorageConfig_ShortS3Key(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	emptyRows := sqlmock.NewRows([]string{"value"}).AddRow("")
	shortKeyRows := sqlmock.NewRows([]string{"value"}).AddRow("short")

	// order of queries in the handler
	keys := []string{
		"storage.driver", "storage.local_path", "storage.s3_endpoint",
		"storage.s3_bucket", "storage.s3_region", "storage.s3_key", "storage.public_url",
	}
	for _, key := range keys {
		if key == "storage.s3_key" {
			mock.ExpectQuery(`SELECT value FROM system_configs`).WithArgs(key).WillReturnRows(shortKeyRows)
		} else {
			mock.ExpectQuery(`SELECT value FROM system_configs`).WithArgs(key).WillReturnRows(emptyRows)
		}
	}

	handler := NewSettingsHandler(sqlxDB)
	r := setupSettingsTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/settings", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})

	s3Key := data["s3_key"].(string)
	assert.Equal(t, "****", s3Key)
}

func TestUpdateStorageConfig_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	// Mock: 7 upserts (no s3_secret since it's empty)
	for i := 0; i < 7; i++ {
		mock.ExpectExec(`INSERT INTO system_configs`).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}

	handler := NewSettingsHandler(sqlxDB)
	r := setupSettingsTestRouter(handler)

	body := `{
		"driver": "local",
		"local_path": "./uploads",
		"s3_endpoint": "",
		"s3_bucket": "",
		"s3_region": "",
		"s3_key": "",
		"s3_secret": "",
		"public_url": ""
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/admin/settings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp["code"])
}

func TestUpdateStorageConfig_WithSecret(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	// Mock: 8 upserts (include s3_secret)
	for i := 0; i < 8; i++ {
		mock.ExpectExec(`INSERT INTO system_configs`).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}

	handler := NewSettingsHandler(sqlxDB)
	r := setupSettingsTestRouter(handler)

	body := `{
		"driver": "s3",
		"local_path": "",
		"s3_endpoint": "https://oss.example.com",
		"s3_bucket": "my-bucket",
		"s3_region": "us-east-1",
		"s3_key": "my-key",
		"s3_secret": "my-secret",
		"public_url": ""
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/admin/settings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateStorageConfig_InvalidDriver(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	handler := NewSettingsHandler(sqlxDB)
	r := setupSettingsTestRouter(handler)

	body := `{
		"driver": "invalid",
		"local_path": "",
		"s3_endpoint": "",
		"s3_bucket": "",
		"s3_region": "",
		"s3_key": "",
		"s3_secret": "",
		"public_url": ""
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/admin/settings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateStorageConfig_InvalidJSON(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	handler := NewSettingsHandler(sqlxDB)
	r := setupSettingsTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/admin/settings", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
