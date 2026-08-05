// Package api T4 公众号管理 CRUD Handler 测试
// 使用 httptest + sqlmock 测试 List/Create/GetByID/Update/Delete
package api

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/weiyeston/weiyeston-v2/internal/repository/account"
)

// t4AccountColumns returns the full column list for wechat_accounts table (for sqlmock rows)
func t4AccountColumns() []string {
	return []string{
		"id", "tenant_id", "name", "wx_original_id", "wx_app_id",
		"wx_app_secret", "auth_type", "auth_status", "refresh_token",
		"access_token", "token_expire_at", "avatar_url", "qr_code_url",
		"description", "fans_count",
		"authorizer_appid", "func_info", "service_type_info", "verify_type_info",
		"nick_name", "head_img", "user_name", "alias",
		"principal_name", "qrcode_url", "signature",
		"deleted_at", "created_at", "updated_at",
	}
}

// t4AccountRowValues returns a full row of test values for sqlmock
func t4AccountRowValues(id, tenantID int64, name string) []driver.Value {
	return []driver.Value{
		id, tenantID, name, nil, nil, nil, int16(1), int16(1), nil, nil, nil, nil, nil,
		nil, 0,
		nil, nil, nil, nil,
		nil, nil, nil, nil,
		nil, nil, nil,
		nil, time.Now(), time.Now(),
	}
}

// ============================================================
// Mock Implementations for T4 Tests
// ============================================================

// testAccountCache 测试用缓存实现（内存缓存，满足 AccountCache 接口）
type testAccountCache struct {
	data map[string]string
}

func newTestAccountCache() *testAccountCache {
	return &testAccountCache{data: make(map[string]string)}
}

func (c *testAccountCache) Del(ctx context.Context, keys ...string) error {
	for _, k := range keys {
		delete(c.data, k)
	}
	return nil
}

// testWechatSvc 测试用微信服务实现，实现 WechatService 接口
type testWechatSvc struct {
	generatePreAuthURLFunc     func(ctx context.Context, tenantID int64) (string, error)
	fetchManualAccessTokenFunc func(ctx context.Context, appID, appSecret string) (string, int, error)
}

func (m *testWechatSvc) GeneratePreAuthURL(ctx context.Context, tenantID int64) (string, error) {
	if m.generatePreAuthURLFunc != nil {
		return m.generatePreAuthURLFunc(ctx, tenantID)
	}
	return "https://test.auth.url", nil
}

func (m *testWechatSvc) FetchManualAccessToken(ctx context.Context, appID, appSecret string) (string, int, error) {
	if m.fetchManualAccessTokenFunc != nil {
		return m.fetchManualAccessTokenFunc(ctx, appID, appSecret)
	}
	return "test_access_token_123", 7200, nil
}

// setupAccountTestRouter 创建测试用的 gin.Engine，注入模拟的中间件数据
func setupAccountTestRouter(h *AccountHandler, tenantID int64, userID int64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenantID)
		c.Set("user_id", userID)
		c.Next()
	})
	return r
}

// ========== T4-1: POST /api/v1/accounts — Create ==========

func TestCreateAccount_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	repo := &account.Repo{DB: sqlxDB}
	wechatSvc := &testWechatSvc{}
	cache := newTestAccountCache()
	logger, _ := zap.NewDevelopment()
	handler := NewAccountHandler(repo, wechatSvc, cache, logger)

	// 期望：唯一性检查（无结果）
	mock.ExpectQuery(`SELECT \* FROM wechat_accounts WHERE wx_app_id = \$1 AND tenant_id = \$2`).
		WithArgs("wx_test_appid", int64(1)).
		WillReturnError(sql.ErrNoRows)

	// 期望：INSERT
	mock.ExpectQuery(`INSERT INTO wechat_accounts`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(int64(123), time.Now(), time.Now()))

	r := setupAccountTestRouter(handler, 1, 1)
	r.POST("/accounts", handler.Create)

	body := `{"name":"test_account_name","wx_app_id":"wx_test_appid","wx_app_secret":"test_secret_123456"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/accounts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])
	assert.NotNil(t, resp["data"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateAccount_ValidationErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("missing required fields", func(t *testing.T) {
		handler := &AccountHandler{}

		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("tenant_id", int64(1))
			c.Next()
		})
		r.POST("/accounts", handler.Create)

		body := `{"wx_app_id":"wx_test","wx_app_secret":"secret"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/accounts", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("appid without wx prefix", func(t *testing.T) {
		handler := &AccountHandler{}

		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("tenant_id", int64(1))
			c.Next()
		})
		r.POST("/accounts", handler.Create)

		body := `{"name":"test_acc","wx_app_id":"abc123","wx_app_secret":"secret"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/accounts", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Contains(t, resp["msg"], "wx")
	})

	t.Run("duplicate appid", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		sqlxDB := sqlx.NewDb(db, "postgres")

		repo := &account.Repo{DB: sqlxDB}
		handler := NewAccountHandler(repo, &testWechatSvc{}, nil, zap.NewNop())

		// 返回已有记录表示 AppId 冲突
		columns := t4AccountColumns()
		rows := sqlmock.NewRows(columns).AddRow(t4AccountRowValues(99, 1, "existing_account")...)
		mock.ExpectQuery(`SELECT \* FROM wechat_accounts WHERE wx_app_id = \$1`).
			WithArgs("wx_duplicate", int64(1)).
			WillReturnRows(rows)

		r := setupAccountTestRouter(handler, 1, 1)
		r.POST("/accounts", handler.Create)

		body := `{"name":"test_name","wx_app_id":"wx_duplicate","wx_app_secret":"secret"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/accounts", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, float64(40901), resp["code"])

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCreateAccount_WechatValidationFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	repo := &account.Repo{DB: sqlxDB}

	// 微信服务返回错误
	wechatSvc := &testWechatSvc{
		fetchManualAccessTokenFunc: func(ctx context.Context, appID, appSecret string) (string, int, error) {
			return "", 0, fmt.Errorf("wechat returned error: invalid appsecret (40125)")
		},
	}
	handler := NewAccountHandler(repo, wechatSvc, nil, zap.NewNop())

	// 唯一性检查无结果
	mock.ExpectQuery(`SELECT \* FROM wechat_accounts WHERE wx_app_id = \$1`).
		WithArgs("wx_invalid", int64(1)).
		WillReturnError(sql.ErrNoRows)

	r := setupAccountTestRouter(handler, 1, 1)
	r.POST("/accounts", handler.Create)

	body := `{"name":"test","wx_app_id":"wx_invalid","wx_app_secret":"wrong_secret"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/accounts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadGateway, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(50201), resp["code"])
	assert.Contains(t, resp["msg"], "40125")

	assert.NoError(t, mock.ExpectationsWereMet())
}

// ========== T4-2: GET /api/v1/accounts — List ==========

func TestListAccounts_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	repo := &account.Repo{DB: sqlxDB}
	handler := NewAccountHandler(repo, &testWechatSvc{}, nil, zap.NewNop())

	// 期望：COUNT
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM wechat_accounts`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(3)))

	// 期望：分页数据
	listCols := []string{"id", "tenant_id", "name", "wx_original_id", "wx_app_id",
		"auth_type", "auth_status", "avatar_url", "qr_code_url",
		"description", "fans_count", "token_expire_at",
		"nick_name", "head_img", "principal_name",
		"created_at", "updated_at"}
	rows := sqlmock.NewRows(listCols).
		AddRow(int64(1), int64(1), "account_a", nil, "wx_aaa", int16(1), int16(1),
			nil, nil, nil, 100, nil, nil, nil, nil, time.Now(), time.Now()).
		AddRow(int64(2), int64(1), "account_b", nil, "wx_bbb", int16(2), int16(1),
			nil, nil, nil, 200, nil, nil, nil, nil, time.Now(), time.Now()).
		AddRow(int64(3), int64(1), "account_c", nil, "wx_ccc", int16(1), int16(2),
			nil, nil, nil, 0, nil, nil, nil, nil, time.Now(), time.Now())

	mock.ExpectQuery(`SELECT id, tenant_id, name, wx_original_id, wx_app_id`).
		WithArgs(int64(1), 20, 0).
		WillReturnRows(rows)

	r := setupAccountTestRouter(handler, 1, 1)
	r.GET("/accounts", handler.List)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/accounts?page=1&page_size=20", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(3), data["total"])
	assert.NotNil(t, data["list"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListAccounts_WithFilters(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	repo := &account.Repo{DB: sqlxDB}
	handler := NewAccountHandler(repo, &testWechatSvc{}, nil, zap.NewNop())

	// 带筛选条件的 COUNT
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM wechat_accounts`).
		WithArgs(int64(1), "%keyword%", 1, 1).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))

	listCols := []string{"id", "tenant_id", "name", "wx_original_id", "wx_app_id",
		"auth_type", "auth_status", "avatar_url", "qr_code_url",
		"description", "fans_count", "token_expire_at",
		"nick_name", "head_img", "principal_name",
		"created_at", "updated_at"}
	rows := sqlmock.NewRows(listCols).
		AddRow(int64(1), int64(1), "test_account", nil, "wx_test", int16(1), int16(1),
			nil, nil, nil, 50, nil, nil, nil, nil, time.Now(), time.Now())

	mock.ExpectQuery(`SELECT id, tenant_id, name, wx_original_id, wx_app_id`).
		WithArgs(int64(1), "%keyword%", 1, 1, 20, 0).
		WillReturnRows(rows)

	r := setupAccountTestRouter(handler, 1, 1)
	r.GET("/accounts", handler.List)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/accounts?page=1&page_size=20&keyword=keyword&auth_type=1&auth_status=1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListAccounts_Unauthenticated(t *testing.T) {
	handler := NewAccountHandler(nil, nil, nil, zap.NewNop())

	r := gin.New()
	r.GET("/accounts", handler.List)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/accounts", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ========== T4-3: GET /api/v1/accounts/:id — GetByID ==========

func TestGetByID_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	repo := &account.Repo{DB: sqlxDB}
	handler := NewAccountHandler(repo, nil, nil, zap.NewNop())

	columns := t4AccountColumns()
	rows := sqlmock.NewRows(columns).AddRow(t4AccountRowValues(1, 1, "test_account")...)
	mock.ExpectQuery(`SELECT \* FROM wechat_accounts WHERE id = \$1`).
		WithArgs(int64(1)).
		WillReturnRows(rows)

	r := setupAccountTestRouter(handler, 1, 1)
	r.GET("/accounts/:id", handler.GetByID)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/accounts/1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(1), data["id"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	repo := &account.Repo{DB: sqlxDB}
	handler := NewAccountHandler(repo, nil, nil, zap.NewNop())

	mock.ExpectQuery(`SELECT \* FROM wechat_accounts WHERE id = \$1`).
		WithArgs(int64(999)).
		WillReturnError(sql.ErrNoRows)

	r := setupAccountTestRouter(handler, 1, 1)
	r.GET("/accounts/:id", handler.GetByID)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/accounts/999", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(40401), resp["code"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetByID_WrongTenant(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	repo := &account.Repo{DB: sqlxDB}
	handler := NewAccountHandler(repo, nil, nil, zap.NewNop())

	// 公众号属于 tenant 2
	columns := t4AccountColumns()
	rows := sqlmock.NewRows(columns).AddRow(t4AccountRowValues(1, 2, "other_tenant_account")...)
	mock.ExpectQuery(`SELECT \* FROM wechat_accounts WHERE id = \$1`).
		WithArgs(int64(1)).
		WillReturnRows(rows)

	// 请求来自 tenant 1
	r := setupAccountTestRouter(handler, 1, 1)
	r.GET("/accounts/:id", handler.GetByID)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/accounts/1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(40301), resp["code"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

// ========== T4-4: PUT /api/v1/accounts/:id — Update ==========

func TestUpdateAccount_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	repo := &account.Repo{DB: sqlxDB}
	handler := NewAccountHandler(repo, &testWechatSvc{}, nil, zap.NewNop())

	name := "original_name"
	appID := "wx_original"
	secret := "original_secret"
	now := time.Now()

	// 查询现有记录
	columns := t4AccountColumns()
	rows := sqlmock.NewRows(columns).AddRow(
		int64(1), int64(1), name, nil, appID, secret, int16(1), int16(1),
		nil, nil, nil, nil, nil, nil, 0,
		nil, nil, nil, nil,
		nil, nil, nil, nil,
		nil, nil, nil,
		nil, now, now,
	)
	mock.ExpectQuery(`SELECT \* FROM wechat_accounts WHERE id = \$1`).
		WithArgs(int64(1)).
		WillReturnRows(rows)

	// UPDATE
	mock.ExpectExec(`UPDATE wechat_accounts SET`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := setupAccountTestRouter(handler, 1, 1)
	r.PUT("/accounts/:id", handler.Update)

	body := `{"name":"updated_name"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/accounts/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateAccount_PlatformRestriction(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	repo := &account.Repo{DB: sqlxDB}
	handler := NewAccountHandler(repo, &testWechatSvc{}, nil, zap.NewNop())

	now := time.Now()
	columns := t4AccountColumns()
	// 平台授权账号 auth_type=2
	rows := sqlmock.NewRows(columns).AddRow(
		int64(1), int64(1), "platform_account", nil, "wx_platform", nil, int16(2), int16(1),
		nil, nil, nil, nil, nil, nil, 0,
		nil, nil, nil, nil,
		nil, nil, nil, nil,
		nil, nil, nil,
		nil, now, now,
	)
	mock.ExpectQuery(`SELECT \* FROM wechat_accounts WHERE id = \$1`).
		WithArgs(int64(1)).
		WillReturnRows(rows)

	r := setupAccountTestRouter(handler, 1, 1)
	r.PUT("/accounts/:id", handler.Update)

	// 尝试修改 AppId
	body := `{"wx_app_id":"wx_new_value"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/accounts/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(40301), resp["code"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateAccount_AppIDConflict(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	repo := &account.Repo{DB: sqlxDB}
	handler := NewAccountHandler(repo, &testWechatSvc{}, nil, zap.NewNop())

	now := time.Now()
	columns := t4AccountColumns()
	// 现有记录
	rows := sqlmock.NewRows(columns).AddRow(
		int64(1), int64(1), "account_a", nil, "wx_orig", nil, int16(1), int16(1),
		nil, nil, nil, nil, nil, nil, 0,
		nil, nil, nil, nil,
		nil, nil, nil, nil,
		nil, nil, nil,
		nil, now, now,
	)
	mock.ExpectQuery(`SELECT \* FROM wechat_accounts WHERE id = \$1`).
		WithArgs(int64(1)).
		WillReturnRows(rows)

	// AppId 唯一性检查冲突
	conflictRows := sqlmock.NewRows(columns).AddRow(t4AccountRowValues(999, 1, "conflict_account")...)
	mock.ExpectQuery(`SELECT \* FROM wechat_accounts WHERE wx_app_id = \$1`).
		WithArgs("wx_conflict", int64(1)).
		WillReturnRows(conflictRows)

	r := setupAccountTestRouter(handler, 1, 1)
	r.PUT("/accounts/:id", handler.Update)

	body := `{"wx_app_id":"wx_conflict"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/accounts/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(40901), resp["code"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

// ========== T4-5: DELETE /api/v1/accounts/:id — Delete ==========

func TestDeleteAccount_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	repo := &account.Repo{DB: sqlxDB}
	cache := newTestAccountCache()
	handler := NewAccountHandler(repo, nil, cache, zap.NewNop())

	now := time.Now()
	columns := t4AccountColumns()
	rows := sqlmock.NewRows(columns).AddRow(
		int64(1), int64(1), "test_account", nil, nil, nil, int16(1), int16(1),
		nil, nil, nil, nil, nil, nil, 0,
		nil, nil, nil, nil,
		nil, nil, nil, nil,
		nil, nil, nil,
		nil, now, now,
	)
	mock.ExpectQuery(`SELECT \* FROM wechat_accounts WHERE id = \$1`).
		WithArgs(int64(1)).
		WillReturnRows(rows)

	// 软删除
	mock.ExpectExec(`UPDATE wechat_accounts SET deleted_at = NOW\(\) WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs(int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := setupAccountTestRouter(handler, 1, 1)
	r.DELETE("/accounts/:id", handler.Delete)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/accounts/1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])
	assert.Equal(t, "已删除", resp["msg"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteAccount_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	repo := &account.Repo{DB: sqlxDB}
	handler := NewAccountHandler(repo, nil, nil, zap.NewNop())

	mock.ExpectQuery(`SELECT \* FROM wechat_accounts WHERE id = \$1`).
		WithArgs(int64(999)).
		WillReturnError(sql.ErrNoRows)

	r := setupAccountTestRouter(handler, 1, 1)
	r.DELETE("/accounts/:id", handler.Delete)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/accounts/999", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(40401), resp["code"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteAccount_WrongTenant(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	repo := &account.Repo{DB: sqlxDB}
	handler := NewAccountHandler(repo, nil, nil, zap.NewNop())

	now := time.Now()
	columns := t4AccountColumns()
	// 公众号属于 tenant 2
	rows := sqlmock.NewRows(columns).AddRow(
		int64(1), int64(2), "other_tenant_account", nil, nil, nil, int16(1), int16(1),
		nil, nil, nil, nil, nil, nil, 0,
		nil, nil, nil, nil,
		nil, nil, nil, nil,
		nil, nil, nil,
		nil, now, now,
	)
	mock.ExpectQuery(`SELECT \* FROM wechat_accounts WHERE id = \$1`).
		WithArgs(int64(1)).
		WillReturnRows(rows)

	r := setupAccountTestRouter(handler, 1, 1)
	r.DELETE("/accounts/:id", handler.Delete)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/accounts/1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(40301), resp["code"])

	assert.NoError(t, mock.ExpectationsWereMet())
}
