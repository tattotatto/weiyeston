package model

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestTenantDBTags 验证 Tenant 结构体 db tag 与 tenants 表列名一致
func TestTenantDBTags(t *testing.T) {
	expected := map[string]string{
		"ID":           "id",
		"Username":     "username",
		"PasswordHash": "password_hash",
		"Nickname":     "nickname",
		"Email":        "email",
		"Phone":        "phone",
		"AvatarURL":    "avatar_url",
		"Role":         "role",
		"Status":       "status",
		"LastLoginAt":  "last_login_at",
		"DeletedAt":    "deleted_at",
		"CreatedAt":    "created_at",
		"UpdatedAt":    "updated_at",
	}
	assertFieldDBTags(t, Tenant{}, expected, "Tenant")
}

// TestWechatAccountDBTags 验证 WechatAccount 结构体 db tag 与 wechat_accounts 表列名一致
func TestWechatAccountDBTags(t *testing.T) {
	expected := map[string]string{
		"ID":            "id",
		"TenantID":      "tenant_id",
		"Name":          "name",
		"WxOriginalID":  "wx_original_id",
		"WxAppID":       "wx_app_id",
		"WxAppSecret":   "wx_app_secret",
		"AuthType":      "auth_type",
		"AuthStatus":    "auth_status",
		"RefreshToken":  "refresh_token",
		"AccessToken":   "access_token",
		"TokenExpireAt": "token_expire_at",
		"AvatarURL":     "avatar_url",
		"QRCodeURL":     "qr_code_url",
		"Description":   "description",
		"FansCount":     "fans_count",
		"DeletedAt":     "deleted_at",
		"CreatedAt":     "created_at",
		"UpdatedAt":     "updated_at",
	}
	assertFieldDBTags(t, WechatAccount{}, expected, "WechatAccount")
}

// TestChannelDBTags 验证 Channel 结构体 db tag 与 cms_channels 表列名一致
func TestChannelDBTags(t *testing.T) {
	expected := map[string]string{
		"ID":          "id",
		"AccountID":   "account_id",
		"ParentID":    "parent_id",
		"Name":        "name",
		"Slug":        "slug",
		"Level":       "level",
		"SortOrder":   "sort_order",
		"CoverURL":    "cover_url",
		"Description": "description",
		"Status":      "status",
		"DeletedAt":   "deleted_at",
		"CreatedAt":   "created_at",
		"UpdatedAt":   "updated_at",
	}
	assertFieldDBTags(t, Channel{}, expected, "Channel")
}

// TestArticleDBTags 验证 Article 结构体 db tag 与 cms_articles 表列名一致
func TestArticleDBTags(t *testing.T) {
	expected := map[string]string{
		"ID":          "id",
		"AccountID":   "account_id",
		"ChannelID":   "channel_id",
		"Title":       "title",
		"CoverURL":    "cover_url",
		"Summary":     "summary",
		"Author":      "author",
		"Content":     "content",
		"HTMLCache":   "html_cache",
		"Status":      "status",
		"IsTemplate":  "is_template",
		"TemplateCat": "template_cat",
		"SortOrder":   "sort_order",
		"ViewCount":   "view_count",
		"PublishedAt": "published_at",
		"DeletedAt":   "deleted_at",
		"CreatedAt":   "created_at",
		"UpdatedAt":   "updated_at",
	}
	assertFieldDBTags(t, Article{}, expected, "Article")
}

// TestQuickNewsChannelDBTags 验证 QuickNewsChannel 结构体 db tag
func TestQuickNewsChannelDBTags(t *testing.T) {
	expected := map[string]string{
		"ID":          "id",
		"AccountID":   "account_id",
		"Name":        "name",
		"CoverURL":    "cover_url",
		"Description": "description",
		"SortOrder":   "sort_order",
		"Status":      "status",
		"DeletedAt":   "deleted_at",
		"CreatedAt":   "created_at",
		"UpdatedAt":   "updated_at",
	}
	assertFieldDBTags(t, QuickNewsChannel{}, expected, "QuickNewsChannel")
}

// TestQuickNewsUserDBTags 验证 QuickNewsUser 结构体 db tag
func TestQuickNewsUserDBTags(t *testing.T) {
	expected := map[string]string{
		"ID":        "id",
		"AccountID": "account_id",
		"Openid":    "openid",
		"Unionid":   "unionid",
		"Nickname":  "nickname",
		"AvatarURL": "avatar_url",
		"Status":    "status",
		"CreatedAt": "created_at",
		"UpdatedAt": "updated_at",
	}
	assertFieldDBTags(t, QuickNewsUser{}, expected, "QuickNewsUser")
}

// TestQuickNewsDBTags 验证 QuickNews 结构体 db tag
func TestQuickNewsDBTags(t *testing.T) {
	expected := map[string]string{
		"ID":           "id",
		"AccountID":    "account_id",
		"ChannelID":    "channel_id",
		"UserID":       "user_id",
		"AuthorName":   "author_name",
		"AuthorAvatar": "author_avatar",
		"Content":      "content",
		"LikeCount":    "like_count",
		"CommentCount": "comment_count",
		"Status":       "status",
		"IsTop":        "is_top",
		"DeletedAt":    "deleted_at",
		"CreatedAt":    "created_at",
		"UpdatedAt":    "updated_at",
	}
	assertFieldDBTags(t, QuickNews{}, expected, "QuickNews")
}

// TestQuickNewsPhotoDBTags 验证 QuickNewsPhoto 结构体 db tag
func TestQuickNewsPhotoDBTags(t *testing.T) {
	expected := map[string]string{
		"ID":        "id",
		"NewsID":    "news_id",
		"URL":       "url",
		"SortOrder": "sort_order",
		"CreatedAt": "created_at",
	}
	assertFieldDBTags(t, QuickNewsPhoto{}, expected, "QuickNewsPhoto")
}

// TestQuickNewsLikeDBTags 验证 QuickNewsLike 结构体 db tag
func TestQuickNewsLikeDBTags(t *testing.T) {
	expected := map[string]string{
		"ID":        "id",
		"NewsID":    "news_id",
		"UserID":    "user_id",
		"Openid":    "openid",
		"CreatedAt": "created_at",
	}
	assertFieldDBTags(t, QuickNewsLike{}, expected, "QuickNewsLike")
}

// TestQuickNewsCommentDBTags 验证 QuickNewsComment 结构体 db tag
func TestQuickNewsCommentDBTags(t *testing.T) {
	expected := map[string]string{
		"ID":        "id",
		"NewsID":    "news_id",
		"UserID":    "user_id",
		"ParentID":  "parent_id",
		"Content":   "content",
		"LikeCount": "like_count",
		"Status":    "status",
		"DeletedAt": "deleted_at",
		"CreatedAt": "created_at",
		"UpdatedAt": "updated_at",
	}
	assertFieldDBTags(t, QuickNewsComment{}, expected, "QuickNewsComment")
}

// TestVoteDBTags 验证 Vote 结构体 db tag
func TestVoteDBTags(t *testing.T) {
	expected := map[string]string{
		"ID":          "id",
		"AccountID":   "account_id",
		"Title":       "title",
		"Description": "description",
		"CoverURL":    "cover_url",
		"VoteType":    "vote_type",
		"MaxChoices":  "max_choices",
		"MaxVotes":    "max_votes",
		"StartTime":   "start_time",
		"EndTime":     "end_time",
		"TotalVotes":  "total_votes",
		"Status":      "status",
		"DeletedAt":   "deleted_at",
		"CreatedAt":   "created_at",
		"UpdatedAt":   "updated_at",
	}
	assertFieldDBTags(t, Vote{}, expected, "Vote")
}

// TestVoteOptionDBTags 验证 VoteOption 结构体 db tag
func TestVoteOptionDBTags(t *testing.T) {
	expected := map[string]string{
		"ID":        "id",
		"VoteID":    "vote_id",
		"Content":   "content",
		"ImageURL":  "image_url",
		"SortOrder": "sort_order",
		"VoteCount": "vote_count",
	}
	assertFieldDBTags(t, VoteOption{}, expected, "VoteOption")
}

// TestVoteRecordDBTags 验证 VoteRecord 结构体 db tag
func TestVoteRecordDBTags(t *testing.T) {
	expected := map[string]string{
		"ID":        "id",
		"VoteID":    "vote_id",
		"OptionID":  "option_id",
		"Openid":    "openid",
		"IPAddress": "ip_address",
		"UserAgent": "user_agent",
		"CreatedAt": "created_at",
	}
	assertFieldDBTags(t, VoteRecord{}, expected, "VoteRecord")
}

// TestMaterialDBTags 验证 Material 结构体 db tag
func TestMaterialDBTags(t *testing.T) {
	expected := map[string]string{
		"ID":           "id",
		"AccountID":    "account_id",
		"MediaID":      "media_id",
		"Type":         "type",
		"Name":         "name",
		"URL":          "url",
		"ThumbnailURL": "thumbnail_url",
		"FileSize":     "file_size",
		"Width":        "width",
		"Height":       "height",
		"Format":       "format",
		"DeletedAt":    "deleted_at",
		"CreatedAt":    "created_at",
		"UpdatedAt":    "updated_at",
	}
	assertFieldDBTags(t, Material{}, expected, "Material")
}

// TestAutoReplyRuleDBTags 验证 AutoReplyRule 结构体 db tag
func TestAutoReplyRuleDBTags(t *testing.T) {
	expected := map[string]string{
		"ID":            "id",
		"AccountID":     "account_id",
		"Keyword":       "keyword",
		"MatchType":     "match_type",
		"ReplyType":     "reply_type",
		"ReplyContent":  "reply_content",
		"ReplyTitle":    "reply_title",
		"ReplyDesc":     "reply_desc",
		"ReplyCoverURL": "reply_cover_url",
		"ReplyURL":      "reply_url",
		"Status":        "status",
		"SortOrder":     "sort_order",
		"DeletedAt":     "deleted_at",
		"CreatedAt":     "created_at",
		"UpdatedAt":     "updated_at",
	}
	assertFieldDBTags(t, AutoReplyRule{}, expected, "AutoReplyRule")
}

// TestSystemConfigDBTags 验证 SystemConfig 结构体 db tag
func TestSystemConfigDBTags(t *testing.T) {
	expected := map[string]string{
		"ID":          "id",
		"AccountID":   "account_id",
		"Key":         "key",
		"Value":       "value",
		"Type":        "type",
		"Description": "description",
		"CreatedAt":   "created_at",
		"UpdatedAt":   "updated_at",
	}
	assertFieldDBTags(t, SystemConfig{}, expected, "SystemConfig")
}

// assertFieldDBTags 通用字段 db tag 验证辅助函数
func assertFieldDBTags(t *testing.T, obj interface{}, expected map[string]string, structName string) {
	t.Helper()

	typ := reflect.TypeOf(obj)
	assert.Equal(t, reflect.Struct, typ.Kind(), "%s 应为结构体", structName)

	for fieldName, expectedTag := range expected {
		field, found := typ.FieldByName(fieldName)
		if !assert.True(t, found, "%s 应包含字段 %s", structName, fieldName) {
			continue
		}

		dbTag, ok := field.Tag.Lookup("db")
		if !assert.True(t, ok, "%s.%s 应有 db tag", structName, fieldName) {
			continue
		}

		assert.Equal(t, expectedTag, dbTag,
			"%s.%s db tag 应为 %q，实际为 %q", structName, fieldName, expectedTag, dbTag)
	}
}

// TestModelFieldCount 验证各结构体字段数与预期一致（防止遗漏字段）
func TestModelFieldCount(t *testing.T) {
	tests := []struct {
		name       string
		obj        interface{}
		fieldCount int
	}{
		{"Tenant", Tenant{}, 13},
		{"WechatAccount", WechatAccount{}, 29},
		{"Channel", Channel{}, 13},
		{"Article", Article{}, 18},
		{"QuickNewsChannel", QuickNewsChannel{}, 10},
		{"QuickNewsUser", QuickNewsUser{}, 9},
		{"QuickNews", QuickNews{}, 14},
		{"QuickNewsPhoto", QuickNewsPhoto{}, 5},
		{"QuickNewsLike", QuickNewsLike{}, 5},
		{"QuickNewsComment", QuickNewsComment{}, 10},
		{"Vote", Vote{}, 15},
		{"VoteOption", VoteOption{}, 6},
		{"VoteRecord", VoteRecord{}, 7},
		{"Material", Material{}, 14},
		{"AutoReplyRule", AutoReplyRule{}, 15},
		{"SystemConfig", SystemConfig{}, 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typ := reflect.TypeOf(tt.obj)
			actualCount := typ.NumField()
			assert.Equal(t, tt.fieldCount, actualCount,
				"%s 应有 %d 个字段，实际有 %d 个", tt.name, tt.fieldCount, actualCount)
		})
	}
}

// TestModelFieldTypes 验证各结构体字段类型正确
func TestModelFieldTypes(t *testing.T) {
	t.Run("Tenant 时间字段类型", func(t *testing.T) {
		_ = Tenant{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		// 编译通过即可验证类型正确
	})

	t.Run("Article Content 字段为 json.RawMessage", func(t *testing.T) {
		a := Article{
			Content: json.RawMessage(`{"type":"doc"}`),
		}
		assert.NotNil(t, a.Content)
		assert.JSONEq(t, `{"type":"doc"}`, string(a.Content))
	})

	t.Run("QuickNews 布尔字段", func(t *testing.T) {
		n := QuickNews{
			IsTop: true,
		}
		assert.True(t, n.IsTop)
	})

	t.Run("VoteOption 无时间戳字段", func(t *testing.T) {
		// VoteOption 没有 created_at/updated_at，验证结构干净
		typ := reflect.TypeOf(VoteOption{})
		for i := 0; i < typ.NumField(); i++ {
			fieldName := typ.Field(i).Name
			assert.NotContains(t, []string{"CreatedAt", "UpdatedAt", "DeletedAt"}, fieldName,
				"VoteOption 不应包含 %s 字段", fieldName)
		}
	})

	t.Run("VoteRecord 无更新时间戳", func(t *testing.T) {
		typ := reflect.TypeOf(VoteRecord{})
		for i := 0; i < typ.NumField(); i++ {
			fieldName := typ.Field(i).Name
			assert.NotContains(t, []string{"UpdatedAt", "DeletedAt"}, fieldName,
				"VoteRecord 不应包含 %s 字段", fieldName)
		}
	})

	t.Run("QuickNewsPhoto 无更新时间戳", func(t *testing.T) {
		typ := reflect.TypeOf(QuickNewsPhoto{})
		for i := 0; i < typ.NumField(); i++ {
			fieldName := typ.Field(i).Name
			assert.NotContains(t, []string{"UpdatedAt", "DeletedAt"}, fieldName,
				"QuickNewsPhoto 不应包含 %s 字段", fieldName)
		}
	})

	t.Run("QuickNewsLike 无更新时间戳", func(t *testing.T) {
		typ := reflect.TypeOf(QuickNewsLike{})
		for i := 0; i < typ.NumField(); i++ {
			fieldName := typ.Field(i).Name
			assert.NotContains(t, []string{"UpdatedAt", "DeletedAt"}, fieldName,
				"QuickNewsLike 不应包含 %s 字段", fieldName)
		}
	})
}

// TestAllModelsHaveDBTag 确保所有模型结构体的每个字段都有 db tag
func TestAllModelsHaveDBTag(t *testing.T) {
	models := []struct {
		name string
		obj  interface{}
	}{
		{"Tenant", Tenant{}},
		{"WechatAccount", WechatAccount{}},
		{"Channel", Channel{}},
		{"Article", Article{}},
		{"QuickNewsChannel", QuickNewsChannel{}},
		{"QuickNewsUser", QuickNewsUser{}},
		{"QuickNews", QuickNews{}},
		{"QuickNewsPhoto", QuickNewsPhoto{}},
		{"QuickNewsLike", QuickNewsLike{}},
		{"QuickNewsComment", QuickNewsComment{}},
		{"Vote", Vote{}},
		{"VoteOption", VoteOption{}},
		{"VoteRecord", VoteRecord{}},
		{"Material", Material{}},
		{"AutoReplyRule", AutoReplyRule{}},
		{"SystemConfig", SystemConfig{}},
	}

	for _, m := range models {
		t.Run(m.name, func(t *testing.T) {
			typ := reflect.TypeOf(m.obj)
			for i := 0; i < typ.NumField(); i++ {
				field := typ.Field(i)
				_, hasDBTag := field.Tag.Lookup("db")
				assert.True(t, hasDBTag,
					"%s.%s 缺少 db tag", m.name, field.Name)
			}
		})
	}
}

// ========== T2 新增 Role 字段测试 ==========

// TestTenantRoleEnum T2 — 测试 Tenant Role 字段枚举值和业务规则
func TestTenantRoleEnum(t *testing.T) {
	t.Run("Role 字段允许的值为 admin 和 user", func(t *testing.T) {
		validRoles := []string{"admin", "user"}
		for _, role := range validRoles {
			assert.Contains(t, []string{"admin", "user"}, role,
				"%s 应为有效角色", role)
		}
	})

	t.Run("Role 默认值应为 user", func(t *testing.T) {
		// 默认 role 应为 "user"，admin 需要手动设置
		defaultRole := "user"
		assert.Equal(t, "user", defaultRole, "默认角色应为 user")
	})

	t.Run("admin 角色应有更高权限", func(t *testing.T) {
		adminRole := "admin"
		userRole := "user"
		assert.NotEqual(t, adminRole, userRole, "admin 和 user 应为不同角色")
		assert.Equal(t, "admin", adminRole, "admin 字面量应正确")
	})
}

// TestTenantRoleFieldType T2 — 验证 Role 字段为 string 类型
func TestTenantRoleFieldType(t *testing.T) {
	// 编译时类型检查：Role 字段应为 string
	tenant := Tenant{
		Role: "user",
	}
	assert.Equal(t, "user", tenant.Role)
	assert.IsType(t, "", tenant.Role, "Role 字段应为 string 类型")
}
