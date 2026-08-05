package testutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// expectedTables 迁移脚本中应包含的所有表名
var expectedTables = []string{
	"tenants",
	"wechat_accounts",
	"auto_reply_rules",
	"cms_channels",
	"cms_articles",
	"quicknews_channels",
	"quicknews_users",
	"quicknews_news",
	"quicknews_photos",
	"quicknews_likes",
	"quicknews_comments",
	"votes",
	"vote_options",
	"vote_records",
	"materials",
	"system_configs",
}

// expectedIndexes 迁移脚本中应包含的关键索引
var expectedIndexes = []string{
	"idx_tenants_username",
	"idx_tenants_status",
	"idx_accounts_tenant",
	"idx_accounts_app_id",
	"idx_accounts_status",
	"idx_auto_reply_account_status",
	"idx_auto_reply_keyword",
	"idx_channels_account",
	"idx_channels_slug",
	"idx_articles_account_status",
	"idx_articles_channel_status",
	"idx_articles_content_gin",
	"idx_articles_title_trgm",
	"idx_articles_templates",
	"idx_qn_channels_account",
	"idx_qn_users_account_openid",
	"idx_qn_users_account",
	"idx_qn_news_channel_status",
	"idx_qn_news_account",
	"idx_qn_news_user",
	"idx_qn_photos_news",
	"idx_qn_likes_news_openid",
	"idx_qn_likes_news",
	"idx_qn_comments_news",
	"idx_qn_comments_user",
	"idx_votes_account_status",
	"idx_votes_time_range",
	"idx_vote_options_vote",
	"idx_vote_records_vote_openid",
	"idx_vote_records_option",
	"idx_materials_account_type",
	"idx_materials_media_id",
	"idx_sys_config_key",
	"idx_sys_config_account",
}

// readMigrationFile 读取迁移脚本文件内容
func readMigrationFile(t *testing.T, filename string) string {
	t.Helper()
	path := filepath.Join("..", "..", "migrations", filename)
	data, err := os.ReadFile(path)
	require.NoError(t, err, "无法读取 %s", filename)
	return string(data)
}

// TestUpSQLContainsAllTables 测试 up.sql 包含所有 16 张表的 CREATE TABLE 语句
func TestUpSQLContainsAllTables(t *testing.T) {
	upSQL := readMigrationFile(t, "000001_init.up.sql")

	for _, table := range expectedTables {
		t.Run("表_"+table, func(t *testing.T) {
			createStmt := "CREATE TABLE IF NOT EXISTS " + table
			assert.Contains(t, upSQL, createStmt,
				"up.sql 应包含 %s 的 CREATE TABLE 语句", table)
		})
	}
}

// TestUpSQLContainsExpectedIndexes 测试 up.sql 包含关键索引
func TestUpSQLContainsExpectedIndexes(t *testing.T) {
	upSQL := readMigrationFile(t, "000001_init.up.sql")

	for _, idx := range expectedIndexes {
		t.Run("索引_"+idx, func(t *testing.T) {
			assert.Contains(t, upSQL, idx,
				"up.sql 应包含索引 %s", idx)
		})
	}
}

// TestUpSQLContainsExtensions 测试 up.sql 开启必要的 PostgreSQL 扩展
func TestUpSQLContainsExtensions(t *testing.T) {
	upSQL := readMigrationFile(t, "000001_init.up.sql")

	t.Run("pgcrypto 扩展", func(t *testing.T) {
		assert.Contains(t, upSQL, `CREATE EXTENSION IF NOT EXISTS "pgcrypto"`,
			"up.sql 应开启 pgcrypto 扩展")
	})

	t.Run("pg_trgm 扩展", func(t *testing.T) {
		assert.Contains(t, upSQL, `CREATE EXTENSION IF NOT EXISTS "pg_trgm"`,
			"up.sql 应开启 pg_trgm 扩展")
	})
}

// TestUpSQLContainsUpdateTimestampFunction 测试 up.sql 包含 updated_at 触发器函数
func TestUpSQLContainsUpdateTimestampFunction(t *testing.T) {
	upSQL := readMigrationFile(t, "000001_init.up.sql")

	t.Run("update_timestamp 函数", func(t *testing.T) {
		assert.Contains(t, upSQL, "CREATE OR REPLACE FUNCTION update_timestamp()",
			"up.sql 应定义 update_timestamp() 函数")
	})
}

// TestUpSQLContainsTriggers 测试 up.sql 为所有业务表创建 updated_at 触发器
func TestUpSQLContainsTriggers(t *testing.T) {
	upSQL := readMigrationFile(t, "000001_init.up.sql")

	tablesWithTrigger := []string{
		"tenants",
		"wechat_accounts",
		"auto_reply_rules",
		"cms_channels",
		"cms_articles",
		"quicknews_channels",
		"quicknews_users",
		"quicknews_news",
		"quicknews_comments",
		"votes",
		"materials",
		"system_configs",
	}

	for _, table := range tablesWithTrigger {
		t.Run("触发器_"+table, func(t *testing.T) {
			// 触发器命名格式: trg_{table}_updated_at
			triggerName := "trg_" + table + "_updated_at"
			assert.Contains(t, upSQL, triggerName,
				"up.sql 应为 %s 表创建 updated_at 触发器", table)
		})
	}
}

// TestDownSQLContainsAllTables 测试 down.sql 包含所有表的 DROP TABLE
func TestDownSQLContainsAllTables(t *testing.T) {
	downSQL := readMigrationFile(t, "000001_init.down.sql")

	for _, table := range expectedTables {
		t.Run("删除表_"+table, func(t *testing.T) {
			dropStmt := "DROP TABLE IF EXISTS " + table
			assert.Contains(t, downSQL, dropStmt,
				"down.sql 应包含 %s 的 DROP TABLE 语句", table)
		})
	}
}

// TestDownSQLDropOrder 测试 down.sql 按外键依赖逆序删除（子表先删，父表后删）
func TestDownSQLDropOrder(t *testing.T) {
	downSQL := readMigrationFile(t, "000001_init.down.sql")

	// 提取 DROP TABLE 语句中的表名，保持顺序
	lines := strings.Split(downSQL, "\n")
	var dropOrder []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "DROP TABLE IF EXISTS ") {
			tableName := strings.TrimPrefix(line, "DROP TABLE IF EXISTS ")
			tableName = strings.TrimSuffix(tableName, ";")
			tableName = strings.TrimSpace(tableName)
			if tableName != "" {
				dropOrder = append(dropOrder, tableName)
			}
		}
	}

	t.Run("至少有 16 个 DROP TABLE 语句", func(t *testing.T) {
		assert.GreaterOrEqual(t, len(dropOrder), 16,
			"down.sql 应包含至少 16 个 DROP TABLE 语句，实际 %d 个", len(dropOrder))
	})

	t.Run("子表先于父表删除", func(t *testing.T) {
		// 验证依赖关系：子表必须先于父表删除
		assertTableBefore := func(child, parent string) {
			t.Helper()
			childIdx := indexOf(dropOrder, child)
			parentIdx := indexOf(dropOrder, parent)
			if childIdx >= 0 && parentIdx >= 0 {
				assert.True(t, childIdx < parentIdx,
					"子表 %s（位置 %d）应在父表 %s（位置 %d）之前删除",
					child, childIdx, parent, parentIdx)
			}
		}

		// 核心依赖链
		assertTableBefore("vote_records", "vote_options")
		assertTableBefore("vote_records", "votes")
		assertTableBefore("vote_options", "votes")

		assertTableBefore("quicknews_comments", "quicknews_news")
		assertTableBefore("quicknews_likes", "quicknews_news")
		assertTableBefore("quicknews_photos", "quicknews_news")

		assertTableBefore("cms_articles", "cms_channels")

		assertTableBefore("auto_reply_rules", "wechat_accounts")

		assertTableBefore("wechat_accounts", "tenants")
	})
}

// TestUpSQLContainsForeignKeyConstraints 测试 up.sql 包含外键约束
func TestUpSQLContainsForeignKeyConstraints(t *testing.T) {
	upSQL := readMigrationFile(t, "000001_init.up.sql")

	expectedFKs := []string{
		"fk_accounts_tenant",
		"fk_auto_reply_account",
		"fk_channels_account",
		"fk_channels_parent",
		"fk_articles_account",
		"fk_articles_channel",
		"fk_qn_channels_account",
		"fk_qn_users_account",
		"fk_qn_news_account",
		"fk_qn_news_channel",
		"fk_qn_news_user",
		"fk_qn_photos_news",
		"fk_qn_likes_news",
		"fk_qn_likes_user",
		"fk_qn_comments_news",
		"fk_qn_comments_user",
		"fk_qn_comments_parent",
		"fk_votes_account",
		"fk_vote_options_vote",
		"fk_vote_records_vote",
		"fk_vote_records_option",
		"fk_materials_account",
		"fk_sys_config_account",
	}

	for _, fk := range expectedFKs {
		t.Run("外键_"+fk, func(t *testing.T) {
			assert.Contains(t, upSQL, fk,
				"up.sql 应包含外键约束 %s", fk)
		})
	}
}

// TestUpSQLHasSoftDelete 测试使用软删除的表有 deleted_at 字段
func TestUpSQLHasSoftDelete(t *testing.T) {
	upSQL := readMigrationFile(t, "000001_init.up.sql")

	softDeleteTables := []string{
		"tenants",
		"wechat_accounts",
		"auto_reply_rules",
		"cms_channels",
		"cms_articles",
		"quicknews_channels",
		"quicknews_news",
		"quicknews_comments",
		"votes",
		"materials",
	}

	for _, table := range softDeleteTables {
		t.Run("软删除_"+table, func(t *testing.T) {
			// 每个软删除表应有 deleted_at 字段
			assert.Contains(t, upSQL, table+" (", "up.sql 应包含表 %s 的定义", table)
			// 在表定义范围内查找 deleted_at
			assert.Contains(t, upSQL, "deleted_at      TIMESTAMPTZ",
				"表 %s 应包含 deleted_at 字段", table)
		})
	}
}

// TestUpSQLHasPartialUniqueIndexes 测试部分唯一索引排除已删除记录
func TestUpSQLHasPartialUniqueIndexes(t *testing.T) {
	upSQL := readMigrationFile(t, "000001_init.up.sql")

	t.Run("tenants username 唯一索引排除已删除", func(t *testing.T) {
		assert.Contains(t, upSQL, "WHERE deleted_at IS NULL",
			"up.sql 应至少有一处 WHERE deleted_at IS NULL 部分索引条件")
	})

	t.Run("wechat_accounts AppId 部分唯一索引", func(t *testing.T) {
		assert.Contains(t, upSQL, "auth_type = 1 AND auth_status IN (0, 1) AND deleted_at IS NULL",
			"up.sql 应包含 AppId 部分唯一索引的条件")
	})
}

// indexOf 返回元素在切片中的索引，未找到返回 -1
func indexOf(slice []string, item string) int {
	for i, s := range slice {
		if s == item {
			return i
		}
	}
	return -1
}
