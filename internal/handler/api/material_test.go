// Package api T7: 素材管理 Handler 测试
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/weiyeston/weiyeston-v2/internal/model"
)

// ========== Mock MaterialRepo ==========

type mockMaterialRepo struct {
	listFn         func(ctx context.Context, accountID int64, materialType string, offset, limit int) ([]model.Material, error)
	getByIDFn      func(ctx context.Context, id int64) (*model.Material, error)
	createFn       func(ctx context.Context, m *model.Material) error
	softDeleteFn   func(ctx context.Context, id int64) (bool, error)
	countFn        func(ctx context.Context, accountID int64, materialType string) (int, error)
}

func (m *mockMaterialRepo) ListByAccount(ctx context.Context, accountID int64, materialType string, offset, limit int) ([]model.Material, error) {
	if m.listFn != nil {
		return m.listFn(ctx, accountID, materialType, offset, limit)
	}
	return make([]model.Material, 0), nil
}

func (m *mockMaterialRepo) GetByID(ctx context.Context, id int64) (*model.Material, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockMaterialRepo) Create(ctx context.Context, mat *model.Material) error {
	if m.createFn != nil {
		return m.createFn(ctx, mat)
	}
	mat.ID = 1
	mat.CreatedAt = time.Now()
	mat.UpdatedAt = time.Now()
	return nil
}

func (m *mockMaterialRepo) SoftDelete(ctx context.Context, id int64) (bool, error) {
	if m.softDeleteFn != nil {
		return m.softDeleteFn(ctx, id)
	}
	return true, nil
}

func (m *mockMaterialRepo) CountByAccount(ctx context.Context, accountID int64, materialType string) (int, error) {
	if m.countFn != nil {
		return m.countFn(ctx, accountID, materialType)
	}
	return 0, nil
}

// ========== Helpers ==========

func setupMaterialTestRouter(handler *MaterialHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", int64(1))
		c.Set("user_id", int64(1))
		c.Next()
	})
	r.GET("/api/v1/materials", handler.List)
	r.POST("/api/v1/materials/upload", handler.Upload)
	r.GET("/api/v1/materials/:id", handler.GetByID)
	r.DELETE("/api/v1/materials/:id", handler.Delete)
	return r
}

func newMaterialHandler(repo MaterialRepo) *MaterialHandler {
	tmpDir := filepath.Join(os.TempDir(), "weiyeston_test_uploads")
	os.MkdirAll(tmpDir, 0755)
	return NewMaterialHandler(repo, tmpDir, zap.NewNop())
}

func sampleMaterials(count int, accountID int64) []model.Material {
	mats := make([]model.Material, 0, count)
	for i := 0; i < count; i++ {
		name := "test.jpg"
		fileSize := int64(102400)
		mats = append(mats, model.Material{
			ID:        int64(i + 1),
			AccountID: accountID,
			Type:      "image",
			Name:      &name,
			URL:       "/uploads/test.jpg",
			FileSize:  &fileSize,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		})
	}
	return mats
}

// ========== List Tests ==========

func TestMaterialList_Success(t *testing.T) {
	mats := sampleMaterials(3, 10)
	mockRepo := &mockMaterialRepo{
		listFn: func(ctx context.Context, accountID int64, materialType string, offset, limit int) ([]model.Material, error) {
			return mats, nil
		},
		countFn: func(ctx context.Context, accountID int64, materialType string) (int, error) {
			return 3, nil
		},
	}

	handler := newMaterialHandler(mockRepo)
	r := setupMaterialTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/materials?account_id=10&page=1&size=20", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(3), data["total"])
	assert.Equal(t, float64(1), data["page"])
}

func TestMaterialList_WithTypeFilter(t *testing.T) {
	mats := sampleMaterials(2, 10)
	mockRepo := &mockMaterialRepo{
		listFn: func(ctx context.Context, accountID int64, materialType string, offset, limit int) ([]model.Material, error) {
			assert.Equal(t, "image", materialType)
			return mats, nil
		},
		countFn: func(ctx context.Context, accountID int64, materialType string) (int, error) {
			return 2, nil
		},
	}

	handler := newMaterialHandler(mockRepo)
	r := setupMaterialTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/materials?account_id=10&type=image", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMaterialList_InvalidAccountID(t *testing.T) {
	handler := newMaterialHandler(&mockMaterialRepo{})
	r := setupMaterialTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/materials?account_id=abc", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMaterialList_MissingAccountID(t *testing.T) {
	handler := newMaterialHandler(&mockMaterialRepo{})
	r := setupMaterialTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/materials", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMaterialList_NoMaterials(t *testing.T) {
	mockRepo := &mockMaterialRepo{
		listFn: func(ctx context.Context, accountID int64, materialType string, offset, limit int) ([]model.Material, error) {
			return make([]model.Material, 0), nil
		},
		countFn: func(ctx context.Context, accountID int64, materialType string) (int, error) {
			return 0, nil
		},
	}

	handler := newMaterialHandler(mockRepo)
	r := setupMaterialTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/materials?account_id=999", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(0), data["total"])
}

func TestMaterialList_RepoError(t *testing.T) {
	mockRepo := &mockMaterialRepo{
		countFn: func(ctx context.Context, accountID int64, materialType string) (int, error) {
			return 0, errors.New("db error")
		},
	}

	handler := newMaterialHandler(mockRepo)
	r := setupMaterialTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/materials?account_id=10", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ========== GetByID Tests ==========

func TestMaterialGetByID_Success(t *testing.T) {
	fileName := "banner.jpg"
	fileSize := int64(204800)
	mockRepo := &mockMaterialRepo{
		getByIDFn: func(ctx context.Context, id int64) (*model.Material, error) {
			return &model.Material{
				ID:        id,
				AccountID: 10,
				Type:      "image",
				Name:      &fileName,
				URL:       "/uploads/banner.jpg",
				FileSize:  &fileSize,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
	}

	handler := newMaterialHandler(mockRepo)
	r := setupMaterialTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/materials/1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(1), data["id"])
}

func TestMaterialGetByID_NotFound(t *testing.T) {
	mockRepo := &mockMaterialRepo{
		getByIDFn: func(ctx context.Context, id int64) (*model.Material, error) {
			return nil, nil
		},
	}

	handler := newMaterialHandler(mockRepo)
	r := setupMaterialTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/materials/999", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestMaterialGetByID_InvalidID(t *testing.T) {
	handler := newMaterialHandler(&mockMaterialRepo{})
	r := setupMaterialTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/materials/abc", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ========== Upload Tests ==========

func TestMaterialUpload_InvalidAccountID(t *testing.T) {
	handler := newMaterialHandler(&mockMaterialRepo{})
	r := setupMaterialTestRouter(handler)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/materials/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMaterialUpload_NoFile(t *testing.T) {
	handler := newMaterialHandler(&mockMaterialRepo{})
	r := setupMaterialTestRouter(handler)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("account_id", "10")
	writer.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/materials/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMaterialUpload_Success(t *testing.T) {
	mockRepo := &mockMaterialRepo{
		createFn: func(ctx context.Context, m *model.Material) error {
			m.ID = 1
			m.CreatedAt = time.Now()
			m.UpdatedAt = time.Now()
			return nil
		},
	}

	handler := newMaterialHandler(mockRepo)
	r := setupMaterialTestRouter(handler)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("account_id", "10")
	part, _ := writer.CreateFormFile("file", "test.png")
	io.WriteString(part, "fake image content")
	writer.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/materials/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp["code"])
	assert.Equal(t, "上传成功", resp["msg"])
}

func TestMaterialUpload_UnsupportedType(t *testing.T) {
	handler := newMaterialHandler(&mockMaterialRepo{})
	r := setupMaterialTestRouter(handler)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("account_id", "10")
	part, _ := writer.CreateFormFile("file", "test.exe")
	io.WriteString(part, "malicious content")
	writer.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/materials/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Contains(t, resp["msg"], "不支持的文件类型")
}

// ========== Delete Tests ==========

func TestMaterialDelete_Success(t *testing.T) {
	mockRepo := &mockMaterialRepo{
		softDeleteFn: func(ctx context.Context, id int64) (bool, error) {
			return true, nil
		},
	}

	handler := newMaterialHandler(mockRepo)
	r := setupMaterialTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/materials/1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])
	assert.Equal(t, "已删除", resp["msg"])
}

func TestMaterialDelete_NotFound(t *testing.T) {
	mockRepo := &mockMaterialRepo{
		softDeleteFn: func(ctx context.Context, id int64) (bool, error) {
			return false, nil
		},
	}

	handler := newMaterialHandler(mockRepo)
	r := setupMaterialTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/materials/999", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestMaterialDelete_InvalidID(t *testing.T) {
	handler := newMaterialHandler(&mockMaterialRepo{})
	r := setupMaterialTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/materials/abc", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMaterialDelete_RepoError(t *testing.T) {
	mockRepo := &mockMaterialRepo{
		softDeleteFn: func(ctx context.Context, id int64) (bool, error) {
			return false, errors.New("db error")
		},
	}

	handler := newMaterialHandler(mockRepo)
	r := setupMaterialTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/materials/1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ========== API Response Format Tests ==========

func TestMaterialAPIResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("normal response has correct content-type", func(t *testing.T) {
		fileName := "img.png"
		mockRepo := &mockMaterialRepo{
			getByIDFn: func(ctx context.Context, id int64) (*model.Material, error) {
				return &model.Material{
					ID:        1,
					AccountID: 10,
					Type:      "image",
					Name:      &fileName,
					URL:       "/uploads/img.png",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}, nil
			},
		}

		handler := newMaterialHandler(mockRepo)
		r := setupMaterialTestRouter(handler)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/materials/1", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	})
}

// ========== MaterialVO Conversion Tests ==========

func TestToMaterialVO_NilValues(t *testing.T) {
	now := time.Now()
	m := &model.Material{
		ID:        1,
		AccountID: 10,
		Type:      "image",
		URL:       "/uploads/test.jpg",
		CreatedAt: now,
		UpdatedAt: now,
	}

	vo := toMaterialVO(m)
	assert.Equal(t, int64(1), vo.ID)
	assert.Equal(t, int64(10), vo.AccountID)
	assert.Equal(t, "image", vo.Type)
	assert.Nil(t, vo.MediaID)
	assert.Nil(t, vo.Name)
	assert.Nil(t, vo.ThumbnailURL)
	assert.Nil(t, vo.FileSize)
	assert.Nil(t, vo.Width)
	assert.Nil(t, vo.Height)
	assert.Nil(t, vo.Format)
}

func TestToMaterialVO_AllValues(t *testing.T) {
	now := time.Now()
	fileName := "banner.jpg"
	mediaID := "wx_123"
	thumbURL := "/uploads/thumb_banner.jpg"
	fileSize := int64(512000)
	width := 1920
	height := 1080
	format := "jpg"

	m := &model.Material{
		ID:           1,
		AccountID:    10,
		MediaID:      &mediaID,
		Type:         "image",
		Name:         &fileName,
		URL:          "/uploads/banner.jpg",
		ThumbnailURL: &thumbURL,
		FileSize:     &fileSize,
		Width:        &width,
		Height:       &height,
		Format:       &format,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	vo := toMaterialVO(m)
	assert.Equal(t, int64(1), vo.ID)
	assert.Equal(t, "wx_123", *vo.MediaID)
	assert.Equal(t, "image", vo.Type)
	assert.Equal(t, "banner.jpg", *vo.Name)
	assert.Equal(t, "/uploads/banner.jpg", vo.URL)
	assert.Equal(t, "/uploads/thumb_banner.jpg", *vo.ThumbnailURL)
	assert.Equal(t, int64(512000), *vo.FileSize)
	assert.Equal(t, 1920, *vo.Width)
	assert.Equal(t, 1080, *vo.Height)
	assert.Equal(t, "jpg", *vo.Format)
}

func TestToMaterialVOs_EmptySlice(t *testing.T) {
	vos := toMaterialVOs([]model.Material{})
	assert.Empty(t, vos)
}

func TestToMaterialVOs_NilSlice(t *testing.T) {
	vos := toMaterialVOs(nil)
	assert.Empty(t, vos)
}

// ========== MaterialRepoAdapter Error Cases ==========

func TestMaterialRepoAdapter_GetByID_InvalidID(t *testing.T) {
	adapter := &materialRepoAdapter{DB: nil}
	m, err := adapter.GetByID(context.Background(), 0)
	assert.Error(t, err)
	assert.Nil(t, m)
	assert.Contains(t, err.Error(), "无效的素材 ID")
}

func TestMaterialRepoAdapter_ListByAccount_InvalidID(t *testing.T) {
	adapter := &materialRepoAdapter{DB: nil}
	_, err := adapter.ListByAccount(context.Background(), 0, "", 0, 20)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "无效的公众号 ID")
}

func TestMaterialRepoAdapter_CountByAccount_InvalidID(t *testing.T) {
	adapter := &materialRepoAdapter{DB: nil}
	_, err := adapter.CountByAccount(context.Background(), 0, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "无效的公众号 ID")
}

// ========== Allowed Extensions Tests ==========

func TestAllowedExtensions(t *testing.T) {
	imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".svg"}
	for _, ext := range imageExts {
		typ, ok := allowedExtensions[ext]
		assert.True(t, ok, "扩展名 %s 应该被允许", ext)
		assert.Equal(t, "image", typ)
	}

	voiceExts := []string{".mp3", ".wav", ".amr"}
	for _, ext := range voiceExts {
		typ, ok := allowedExtensions[ext]
		assert.True(t, ok, "扩展名 %s 应该被允许", ext)
		assert.Equal(t, "voice", typ)
	}

	_, ok := allowedExtensions[".exe"]
	assert.False(t, ok, ".exe 不应被允许")
}

// ========== MaterialRepoInterface Tests ==========

func TestMaterialRepoInterface(t *testing.T) {
	var _ MaterialRepo = &materialRepoAdapter{}
	assert.True(t, true, "materialRepoAdapter implements MaterialRepo")
}

// ========== Edge Case: Upload with repo create error ==========

func TestMaterialUpload_CreateError(t *testing.T) {
	mockRepo := &mockMaterialRepo{
		createFn: func(ctx context.Context, m *model.Material) error {
			return errors.New("db insert failed")
		},
	}

	handler := newMaterialHandler(mockRepo)
	r := setupMaterialTestRouter(handler)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("account_id", "10")
	part, _ := writer.CreateFormFile("file", "test.jpg")
	io.WriteString(part, "fake jpg content")
	writer.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/materials/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ========== Edge Case: JPG upload ==========

func TestMaterialUpload_JPGFile(t *testing.T) {
	mockRepo := &mockMaterialRepo{}

	handler := newMaterialHandler(mockRepo)
	r := setupMaterialTestRouter(handler)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("account_id", "10")
	part, _ := writer.CreateFormFile("file", "photo.jpg")
	io.WriteString(part, "fake jpg data")
	writer.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/materials/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "image", data["type"])
	assert.Equal(t, "jpg", data["format"])
}

// ========== Edge Case: PDF upload ==========

func TestMaterialUpload_PDFFile(t *testing.T) {
	mockRepo := &mockMaterialRepo{}

	handler := newMaterialHandler(mockRepo)
	r := setupMaterialTestRouter(handler)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("account_id", "10")
	part, _ := writer.CreateFormFile("file", "document.pdf")
	io.WriteString(part, "fake pdf data")
	writer.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/materials/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "file", data["type"])
	assert.Equal(t, "pdf", data["format"])
}

// ========== Pagination Tests ==========

func TestMaterialList_Pagination(t *testing.T) {
	allMats := sampleMaterials(5, 10)
	mockRepo := &mockMaterialRepo{
		listFn: func(ctx context.Context, accountID int64, materialType string, offset, limit int) ([]model.Material, error) {
			return allMats[offset:min(offset+limit, len(allMats))], nil
		},
		countFn: func(ctx context.Context, accountID int64, materialType string) (int, error) {
			return 5, nil
		},
	}

	handler := newMaterialHandler(mockRepo)
	r := setupMaterialTestRouter(handler)

	// Page 1, size 2
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/materials?account_id=10&page=1&size=2", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(5), data["total"])
	list := data["list"].([]interface{})
	assert.Len(t, list, 2)
}
