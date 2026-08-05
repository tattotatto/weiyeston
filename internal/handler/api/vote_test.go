// Package api T14: 投票管理 Handler 测试
package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/weiyeston/weiyeston-v2/internal/model"
)

// ========== Mock VoteRepo ==========

type mockVoteRepo struct {
	listVotesFn           func(ctx context.Context, accountID int64, offset, limit int) ([]model.Vote, error)
	getVoteFn             func(ctx context.Context, id int64) (*model.Vote, error)
	createVoteFn          func(ctx context.Context, v *model.Vote) error
	updateVoteFn          func(ctx context.Context, v *model.Vote) error
	softDeleteVoteFn      func(ctx context.Context, id int64) error
	getOptionsFn          func(ctx context.Context, voteID int64) ([]model.VoteOption, error)
	createOptionFn        func(ctx context.Context, o *model.VoteOption) error
	deleteOptionsByVoteIDFn func(ctx context.Context, voteID int64) error
	submitVoteFn          func(ctx context.Context, record *model.VoteRecord) error
	countVotesByUserFn    func(ctx context.Context, voteID int64, openid string) (int, error)
	getResultsFn          func(ctx context.Context, voteID int64) ([]model.VoteOption, error)
}

func (m *mockVoteRepo) ListVotes(ctx context.Context, accountID int64, offset, limit int) ([]model.Vote, error) {
	if m.listVotesFn != nil {
		return m.listVotesFn(ctx, accountID, offset, limit)
	}
	return nil, nil
}

func (m *mockVoteRepo) GetVote(ctx context.Context, id int64) (*model.Vote, error) {
	if m.getVoteFn != nil {
		return m.getVoteFn(ctx, id)
	}
	return nil, nil
}

func (m *mockVoteRepo) CreateVote(ctx context.Context, v *model.Vote) error {
	if m.createVoteFn != nil {
		return m.createVoteFn(ctx, v)
	}
	v.ID = 1
	v.CreatedAt = time.Now()
	v.UpdatedAt = time.Now()
	return nil
}

func (m *mockVoteRepo) UpdateVote(ctx context.Context, v *model.Vote) error {
	if m.updateVoteFn != nil {
		return m.updateVoteFn(ctx, v)
	}
	return nil
}

func (m *mockVoteRepo) SoftDeleteVote(ctx context.Context, id int64) error {
	if m.softDeleteVoteFn != nil {
		return m.softDeleteVoteFn(ctx, id)
	}
	return nil
}

func (m *mockVoteRepo) GetOptions(ctx context.Context, voteID int64) ([]model.VoteOption, error) {
	if m.getOptionsFn != nil {
		return m.getOptionsFn(ctx, voteID)
	}
	return nil, nil
}

func (m *mockVoteRepo) CreateOption(ctx context.Context, o *model.VoteOption) error {
	if m.createOptionFn != nil {
		return m.createOptionFn(ctx, o)
	}
	o.ID = 1
	return nil
}

func (m *mockVoteRepo) DeleteOptionsByVoteID(ctx context.Context, voteID int64) error {
	if m.deleteOptionsByVoteIDFn != nil {
		return m.deleteOptionsByVoteIDFn(ctx, voteID)
	}
	return nil
}

func (m *mockVoteRepo) SubmitVote(ctx context.Context, record *model.VoteRecord) error {
	if m.submitVoteFn != nil {
		return m.submitVoteFn(ctx, record)
	}
	record.ID = 1
	record.CreatedAt = time.Now()
	return nil
}

func (m *mockVoteRepo) CountVotesByUser(ctx context.Context, voteID int64, openid string) (int, error) {
	if m.countVotesByUserFn != nil {
		return m.countVotesByUserFn(ctx, voteID, openid)
	}
	return 0, nil
}

func (m *mockVoteRepo) GetResults(ctx context.Context, voteID int64) ([]model.VoteOption, error) {
	if m.getResultsFn != nil {
		return m.getResultsFn(ctx, voteID)
	}
	return nil, nil
}

// ========== Helpers ==========

func setupVoteTestRouter(handler *VoteHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Auth middleware for API routes
	authGroup := r.Group("/api/v1")
	authGroup.Use(func(c *gin.Context) {
		c.Set("tenant_id", int64(1))
		c.Set("user_id", int64(1))
		c.Next()
	})
	{
		authGroup.GET("/votes", handler.ListVotes)
		authGroup.POST("/votes", handler.CreateVote)
		authGroup.GET("/votes/:id", handler.GetVote)
		authGroup.PUT("/votes/:id", handler.UpdateVote)
		authGroup.DELETE("/votes/:id", handler.DeleteVote)
		authGroup.GET("/votes/:id/results", handler.GetResults)
	}

	// H5 routes (no auth)
	r.POST("/h5/vote/:id/submit", handler.SubmitVote)

	return r
}

func newVoteHandler(repo VoteRepo) *VoteHandler {
	return NewVoteHandler(repo, zap.NewNop())
}

// ======================== List Tests ========================

func TestVote_ListVotes_Success(t *testing.T) {
	mockRepo := &mockVoteRepo{
		listVotesFn: func(ctx context.Context, accountID int64, offset, limit int) ([]model.Vote, error) {
			return []model.Vote{
				{ID: 1, AccountID: 1, Title: "测试投票", Status: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()},
			}, nil
		},
	}

	handler := newVoteHandler(mockRepo)
	r := setupVoteTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/votes?page=1&size=10", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp["code"])
}

// ======================== Create Tests ========================

func TestVote_CreateVote_Success(t *testing.T) {
	mockRepo := &mockVoteRepo{
		getOptionsFn: func(ctx context.Context, voteID int64) ([]model.VoteOption, error) {
			return []model.VoteOption{
				{ID: 1, VoteID: voteID, Content: "选项A", SortOrder: 0, VoteCount: 0},
			}, nil
		},
	}
	handler := newVoteHandler(mockRepo)
	r := setupVoteTestRouter(handler)

	body := `{"title": "最佳员工评选", "vote_type": 1, "status": 1, "options": [{"content": "张三", "sort_order": 0}, {"content": "李四", "sort_order": 1}]}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/votes", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp["code"])
	assert.Equal(t, "投票已创建", resp["msg"])
}

func TestVote_CreateVote_InvalidJSON(t *testing.T) {
	handler := newVoteHandler(&mockVoteRepo{})
	r := setupVoteTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/votes", strings.NewReader("bad"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVote_CreateVote_NoOptions(t *testing.T) {
	handler := newVoteHandler(&mockVoteRepo{})
	r := setupVoteTestRouter(handler)

	body := `{"title": "无选项投票"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/votes", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ======================== Get Tests ========================

func TestVote_GetVote_Success(t *testing.T) {
	mockRepo := &mockVoteRepo{
		getVoteFn: func(ctx context.Context, id int64) (*model.Vote, error) {
			return &model.Vote{
				ID: id, AccountID: 1, Title: "测试投票",
				Status: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}, nil
		},
		getOptionsFn: func(ctx context.Context, voteID int64) ([]model.VoteOption, error) {
			return []model.VoteOption{
				{ID: 1, VoteID: voteID, Content: "选项A", VoteCount: 5},
			}, nil
		},
	}

	handler := newVoteHandler(mockRepo)
	r := setupVoteTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/votes/1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestVote_GetVote_NotFound(t *testing.T) {
	mockRepo := &mockVoteRepo{
		getVoteFn: func(ctx context.Context, id int64) (*model.Vote, error) {
			return nil, nil
		},
	}

	handler := newVoteHandler(mockRepo)
	r := setupVoteTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/votes/999", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ======================== Update Tests ========================

func TestVote_UpdateVote_Success(t *testing.T) {
	mockRepo := &mockVoteRepo{}
	handler := newVoteHandler(mockRepo)
	r := setupVoteTestRouter(handler)

	body := `{"title": "更新后的标题", "status": 1}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/votes/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])
	assert.Equal(t, "投票已更新", resp["msg"])
}

// ======================== Delete Tests ========================

func TestVote_DeleteVote_Success(t *testing.T) {
	mockRepo := &mockVoteRepo{}
	handler := newVoteHandler(mockRepo)
	r := setupVoteTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/votes/1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])
	assert.Equal(t, "投票已删除", resp["msg"])
}

// ======================== Results Tests ========================

func TestVote_GetResults_Success(t *testing.T) {
	mockRepo := &mockVoteRepo{
		getVoteFn: func(ctx context.Context, id int64) (*model.Vote, error) {
			return &model.Vote{
				ID: id, AccountID: 1, Title: "投票结果",
				Status: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}, nil
		},
		getResultsFn: func(ctx context.Context, voteID int64) ([]model.VoteOption, error) {
			return []model.VoteOption{
				{ID: 1, VoteID: voteID, Content: "选项A", VoteCount: 10},
				{ID: 2, VoteID: voteID, Content: "选项B", VoteCount: 5},
			}, nil
		},
	}

	handler := newVoteHandler(mockRepo)
	r := setupVoteTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/votes/1/results", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp["code"])
}

// ======================== SubmitVote Tests ========================

func TestVote_SubmitVote_Success(t *testing.T) {
	mockRepo := &mockVoteRepo{
		getVoteFn: func(ctx context.Context, id int64) (*model.Vote, error) {
			return &model.Vote{
				ID: id, AccountID: 1, Title: "投票", Status: 1,
				VoteType: 1, MaxChoices: 1, MaxVotes: 1,
				CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}, nil
		},
		countVotesByUserFn: func(ctx context.Context, voteID int64, openid string) (int, error) {
			return 0, nil
		},
	}

	handler := newVoteHandler(mockRepo)
	r := setupVoteTestRouter(handler)

	body := `{"option_ids": [1], "openid": "test_openid_001"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/h5/vote/1/submit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp["code"])
	assert.Equal(t, "投票成功", resp["msg"])
}

func TestVote_SubmitVote_NotRunning(t *testing.T) {
	mockRepo := &mockVoteRepo{
		getVoteFn: func(ctx context.Context, id int64) (*model.Vote, error) {
			return &model.Vote{
				ID: id, AccountID: 1, Title: "已结束投票", Status: 2,
				CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}, nil
		},
	}

	handler := newVoteHandler(mockRepo)
	r := setupVoteTestRouter(handler)

	body := `{"option_ids": [1], "openid": "test_openid"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/h5/vote/1/submit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVote_SubmitVote_MaxVotesReached(t *testing.T) {
	mockRepo := &mockVoteRepo{
		getVoteFn: func(ctx context.Context, id int64) (*model.Vote, error) {
			return &model.Vote{
				ID: id, AccountID: 1, Title: "投票", Status: 1,
				VoteType: 1, MaxChoices: 1, MaxVotes: 1,
				CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}, nil
		},
		countVotesByUserFn: func(ctx context.Context, voteID int64, openid string) (int, error) {
			return 1, nil // Already voted max times
		},
	}

	handler := newVoteHandler(mockRepo)
	r := setupVoteTestRouter(handler)

	body := `{"option_ids": [1], "openid": "test_openid"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/h5/vote/1/submit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVote_SubmitVote_TimeValidation(t *testing.T) {
	future := time.Now().Add(24 * time.Hour)
	mockRepo := &mockVoteRepo{
		getVoteFn: func(ctx context.Context, id int64) (*model.Vote, error) {
			return &model.Vote{
				ID: id, AccountID: 1, Title: "未来投票", Status: 1,
				StartTime: &future, VoteType: 1, MaxChoices: 1, MaxVotes: 1,
				CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}, nil
		},
	}

	handler := newVoteHandler(mockRepo)
	r := setupVoteTestRouter(handler)

	body := `{"option_ids": [1], "openid": "test_openid"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/h5/vote/1/submit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVote_SubmitVote_InvalidJSON(t *testing.T) {
	handler := newVoteHandler(&mockVoteRepo{})
	r := setupVoteTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/h5/vote/1/submit", strings.NewReader("bad"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ======================== 未认证测试 ========================

func TestVote_APIRoutes_Unauthenticated(t *testing.T) {
	handler := newVoteHandler(&mockVoteRepo{})
	r := gin.New()
	r.GET("/api/v1/votes", handler.ListVotes)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/votes", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ======================== 错误处理测试 ========================

func TestVote_ListVotes_RepoError(t *testing.T) {
	mockRepo := &mockVoteRepo{
		listVotesFn: func(ctx context.Context, accountID int64, offset, limit int) ([]model.Vote, error) {
			return nil, errors.New("database error")
		},
	}

	handler := newVoteHandler(mockRepo)
	r := setupVoteTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/votes", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ======================== 接口满足性测试 ========================

func TestVoteRepoInterface(t *testing.T) {
	var _ VoteRepo = (*mockVoteRepo)(nil)
	assert.True(t, true, "mockVoteRepo implements VoteRepo")
}

// ======================== VO 转换测试 ========================

func TestVoteVO_StatusText(t *testing.T) {
	tests := []struct {
		status     int16
		statusText string
	}{
		{0, "草稿"},
		{1, "进行中"},
		{2, "已结束"},
	}

	for _, tt := range tests {
		v := &model.Vote{ID: 1, Title: "test", Status: tt.status, CreatedAt: time.Now(), UpdatedAt: time.Now()}
		vo := toVoteVO(v, nil)
		assert.Equal(t, tt.statusText, vo.StatusText, "status %d should map to %s", tt.status, tt.statusText)
	}
}

// ======================== Submit with multiple options ========================

func TestVote_SubmitVote_MultipleOptions(t *testing.T) {
	mockRepo := &mockVoteRepo{
		getVoteFn: func(ctx context.Context, id int64) (*model.Vote, error) {
			return &model.Vote{
				ID: id, AccountID: 1, Title: "多选投票", Status: 1,
				VoteType: 2, MaxChoices: 3, MaxVotes: 2,
				CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}, nil
		},
		countVotesByUserFn: func(ctx context.Context, voteID int64, openid string) (int, error) {
			return 0, nil
		},
	}

	handler := newVoteHandler(mockRepo)
	r := setupVoteTestRouter(handler)

	body := `{"option_ids": [1, 2, 3], "openid": "test_openid"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/h5/vote/1/submit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestVote_SubmitVote_SingleSelectButMultipleOptions(t *testing.T) {
	mockRepo := &mockVoteRepo{
		getVoteFn: func(ctx context.Context, id int64) (*model.Vote, error) {
			return &model.Vote{
				ID: id, AccountID: 1, Title: "单选投票", Status: 1,
				VoteType: 1, MaxChoices: 1, MaxVotes: 1,
				CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}, nil
		},
	}

	handler := newVoteHandler(mockRepo)
	r := setupVoteTestRouter(handler)

	body := `{"option_ids": [1, 2], "openid": "test_openid"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/h5/vote/1/submit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
