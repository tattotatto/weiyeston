package testutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigrationFilesExist 测试迁移文件存在且配对
func TestMigrationFilesExist(t *testing.T) {
	t.Run("migrations 目录存在", func(t *testing.T) {
		migrationsDir := filepath.Join("..", "..", "migrations")
		info, err := os.Stat(migrationsDir)
		require.NoError(t, err, "migrations 目录应存在")
		assert.True(t, info.IsDir(), "migrations 应为目录")
	})

	t.Run("存在 000001_init.up.sql", func(t *testing.T) {
		upPath := filepath.Join("..", "..", "migrations", "000001_init.up.sql")
		_, err := os.Stat(upPath)
		assert.NoError(t, err, "000001_init.up.sql 应存在")
	})

	t.Run("存在 000001_init.down.sql", func(t *testing.T) {
		downPath := filepath.Join("..", "..", "migrations", "000001_init.down.sql")
		_, err := os.Stat(downPath)
		assert.NoError(t, err, "000001_init.down.sql 应存在")
	})
}

// TestMigrationFilesPaired 测试 up/down SQL 文件配对
func TestMigrationFilesPaired(t *testing.T) {
	t.Run("每个 up 文件都有对应的 down 文件", func(t *testing.T) {
		migrationsDir := filepath.Join("..", "..", "migrations")
		entries, err := os.ReadDir(migrationsDir)
		require.NoError(t, err)

		upFiles := make(map[string]bool)
		downFiles := make(map[string]bool)

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if strings.HasSuffix(name, ".up.sql") {
				prefix := strings.TrimSuffix(name, ".up.sql")
				upFiles[prefix] = true
			}
			if strings.HasSuffix(name, ".down.sql") {
				prefix := strings.TrimSuffix(name, ".down.sql")
				downFiles[prefix] = true
			}
		}

		for prefix := range upFiles {
			assert.True(t, downFiles[prefix],
				"up 文件 %s.up.sql 应有对应的 down 文件", prefix)
		}

		for prefix := range downFiles {
			assert.True(t, upFiles[prefix],
				"down 文件 %s.down.sql 应有对应的 up 文件", prefix)
		}
	})

	t.Run("迁移文件遵循命名规范", func(t *testing.T) {
		migrationsDir := filepath.Join("..", "..", "migrations")
		entries, err := os.ReadDir(migrationsDir)
		require.NoError(t, err)

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if strings.HasSuffix(name, ".sql") {
				// 格式: {6位序号}_{描述}.{direction}.sql
				parts := strings.SplitN(name, "_", 2)
				if len(parts) == 2 {
					assert.Len(t, parts[0], 6, "序号应为 6 位数字: %s", name)
				}
			}
		}
	})
}

// TestMigrationFileContent 测试迁移文件内容非空
func TestMigrationFileContent(t *testing.T) {
	t.Run("000001_init.up.sql 内容非空", func(t *testing.T) {
		upPath := filepath.Join("..", "..", "migrations", "000001_init.up.sql")
		data, err := os.ReadFile(upPath)
		require.NoError(t, err)
		assert.NotEmpty(t, data, "up.sql 文件内容不应为空")
		assert.Greater(t, len(data), 100, "初始迁移文件应包含较多内容")
	})

	t.Run("000001_init.down.sql 内容非空", func(t *testing.T) {
		downPath := filepath.Join("..", "..", "migrations", "000001_init.down.sql")
		data, err := os.ReadFile(downPath)
		require.NoError(t, err)
		assert.NotEmpty(t, data, "down.sql 文件内容不应为空")
		assert.Greater(t, len(data), 100, "回滚脚本应包含较多内容")
	})
}
