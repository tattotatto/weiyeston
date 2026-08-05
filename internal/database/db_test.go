// Package database 数据库连接与连接池管理
package database

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newMockDB 创建 mock 数据库连接（sqlx 封装）
func newMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err, "创建 sqlmock 失败")
	sqlxDB := sqlx.NewDb(db, "postgres")
	return sqlxDB, mock
}

// newMockDBWithPing 创建支持 Ping 监控的 mock 数据库连接
func newMockDBWithPing(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err, "创建 sqlmock 失败")
	sqlxDB := sqlx.NewDb(db, "postgres")
	return sqlxDB, mock
}

// TestConnectionPoolConfig 测试连接池配置
func TestConnectionPoolConfig(t *testing.T) {
	t.Run("设置最大连接数", func(t *testing.T) {
		sqlxDB, _ := newMockDB(t)
		defer sqlxDB.Close()

		sqlxDB.SetMaxOpenConns(25)
		_ = sqlxDB.Stats()
		assert.NotNil(t, sqlxDB)
	})

	t.Run("设置最大空闲连接数", func(t *testing.T) {
		sqlxDB, _ := newMockDB(t)
		defer sqlxDB.Close()

		sqlxDB.SetMaxIdleConns(5)
		assert.NotNil(t, sqlxDB)
	})

	t.Run("设置连接最大生命周期", func(t *testing.T) {
		sqlxDB, _ := newMockDB(t)
		defer sqlxDB.Close()

		sqlxDB.SetConnMaxLifetime(5 * time.Minute)
		assert.NotNil(t, sqlxDB)
	})

	t.Run("设置连接最大空闲时间", func(t *testing.T) {
		sqlxDB, _ := newMockDB(t)
		defer sqlxDB.Close()

		sqlxDB.SetConnMaxIdleTime(1 * time.Minute)
		assert.NotNil(t, sqlxDB)
	})
}

// TestConnectionPoolDefaultValues 测试连接池默认值配置
func TestConnectionPoolDefaultValues(t *testing.T) {
	tests := []struct {
		name         string
		maxOpenConns int
		maxIdleConns int
		maxLifetime  time.Duration
	}{
		{
			name:         "开发环境默认配置",
			maxOpenConns: 25,
			maxIdleConns: 5,
			maxLifetime:  5 * time.Minute,
		},
		{
			name:         "低配配置",
			maxOpenConns: 5,
			maxIdleConns: 1,
			maxLifetime:  3 * time.Minute,
		},
		{
			name:         "高并发配置",
			maxOpenConns: 100,
			maxIdleConns: 20,
			maxLifetime:  30 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sqlxDB, _ := newMockDB(t)
			defer sqlxDB.Close()

			sqlxDB.SetMaxOpenConns(tt.maxOpenConns)
			sqlxDB.SetMaxIdleConns(tt.maxIdleConns)
			sqlxDB.SetConnMaxLifetime(tt.maxLifetime)

			// 验证配置后 Stats 返回有效的结构体
			stats := sqlxDB.Stats()
			assert.IsType(t, sql.DBStats{}, stats)
			assert.GreaterOrEqual(t, int(stats.MaxOpenConnections), 0,
				"MaxOpenConnections 应为非负数")
		})
	}
}

// TestDBPingSuccess 测试 Ping 成功
func TestDBPingSuccess(t *testing.T) {
	sqlxDB, mock := newMockDBWithPing(t)
	defer sqlxDB.Close()

	mock.ExpectPing()

	err := sqlxDB.Ping()
	assert.NoError(t, err, "Ping 成功不应返回错误")

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestDBPingFailure 测试 Ping 失败
func TestDBPingFailure(t *testing.T) {
	t.Run("Ping 返回连接拒绝错误", func(t *testing.T) {
		sqlxDB, mock := newMockDBWithPing(t)
		defer sqlxDB.Close()

		mock.ExpectPing().WillReturnError(errors.New("connection refused"))

		err := sqlxDB.Ping()
		assert.Error(t, err, "Ping 失败应返回错误")
		assert.Contains(t, err.Error(), "connection refused")

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Ping 返回超时错误", func(t *testing.T) {
		sqlxDB, mock := newMockDBWithPing(t)
		defer sqlxDB.Close()

		mock.ExpectPing().WillReturnError(errors.New("i/o timeout"))

		err := sqlxDB.Ping()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout")

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestDBClose 测试数据库连接关闭
func TestDBClose(t *testing.T) {
	t.Run("正常关闭连接", func(t *testing.T) {
		sqlxDB, mock := newMockDB(t)
		mock.ExpectClose()

		err := sqlxDB.Close()
		assert.NoError(t, err, "关闭连接不应返回错误")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestDBStats 测试连接池统计信息
func TestDBStats(t *testing.T) {
	t.Run("新创建的连接池统计信息", func(t *testing.T) {
		sqlxDB, _ := newMockDB(t)
		defer sqlxDB.Close()

		stats := sqlxDB.Stats()

		assert.GreaterOrEqual(t, stats.OpenConnections, 0,
			"新连接池 OpenConnections 应非负")
		assert.GreaterOrEqual(t, stats.InUse, 0,
			"新连接池 InUse 应非负")
		assert.GreaterOrEqual(t, stats.Idle, 0,
			"新连接池 Idle 应非负")
	})

	t.Run("Stats 返回非 nil 结构体", func(t *testing.T) {
		sqlxDB, _ := newMockDB(t)
		defer sqlxDB.Close()

		sqlxDB.SetMaxOpenConns(10)
		sqlxDB.SetMaxIdleConns(3)

		stats := sqlxDB.Stats()
		assert.IsType(t, sql.DBStats{}, stats)
	})
}

// TestPingWithContext 测试带 Context 的 Ping
func TestPingWithContext(t *testing.T) {
	t.Run("Context Ping 成功", func(t *testing.T) {
		sqlxDB, mock := newMockDBWithPing(t)
		defer sqlxDB.Close()

		mock.ExpectPing()

		err := sqlxDB.PingContext(context.Background())
		assert.NoError(t, err, "PingContext 成功不应返回错误")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Context Ping 失败", func(t *testing.T) {
		sqlxDB, mock := newMockDBWithPing(t)
		defer sqlxDB.Close()

		mock.ExpectPing().WillReturnError(errors.New("context deadline exceeded"))

		err := sqlxDB.PingContext(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
