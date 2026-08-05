// Package database 数据库迁移管理
package database

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
)

// RunMigrations 执行数据库迁移
// 按文件名排序读取 migrationsPath 目录下的 .up.sql 文件并顺序执行
func RunMigrations(db *sqlx.DB, migrationsPath string) error {
	entries, err := os.ReadDir(migrationsPath)
	if err != nil {
		return fmt.Errorf("读取迁移目录失败: %w", err)
	}

	// 收集所有 .up.sql 文件并排序
	var upFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".up.sql") {
			upFiles = append(upFiles, name)
		}
	}
	sort.Strings(upFiles)

	// 顺序执行每个迁移文件
	for _, fileName := range upFiles {
		filePath := filepath.Join(migrationsPath, fileName)
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("读取迁移文件 %s 失败: %w", fileName, err)
		}

		if _, err := db.Exec(string(data)); err != nil {
			return fmt.Errorf("执行迁移 %s 失败: %w", fileName, err)
		}
	}

	return nil
}
