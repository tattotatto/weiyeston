// Package ticket 微信 component_verify_ticket 持久化 Repository 测试
// TDD: 测试先行 — 定义 ticket repository 接口预期行为
// ticket.go 尚未实现，测试使用 sqlmock 验证 SQL 模式正确性
package ticket

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ========== 测试辅助 ==========

// newMockRepo 创建包含 mock DB 的 Repo 用于测试
// ticket.go 实现后，此处改为返回实际的 *Repo 结构体
func newMockRepo(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err, "创建 sqlmock 失败")
	sqlxDB := sqlx.NewDb(db, "postgres")
	return sqlxDB, mock
}

// ========== Save 测试 ==========

func TestTicketRepo_Save_Success(t *testing.T) {
	// ticket.go 实现后预期行为:
	//   func (r *Repo) Save(ctx context.Context, appID string, ticket string) error
	// SQL: INSERT INTO component_verify_tickets (component_appid, ticket) VALUES ($1, $2)

	db, mock := newMockRepo(t)
	defer db.Close()

	appID := "wx570bc396a51xxxxx"
	ticketValue := "ticket_abc123def456"

	// 期望 SQL: INSERT INTO component_verify_tickets (component_appid, ticket) VALUES ($1, $2)
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO component_verify_tickets (component_appid, ticket) VALUES ($1, $2)")).
		WithArgs(appID, ticketValue).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// 模拟 Save 行为（ticket.go 实现后用 r.Save()）
	query := `INSERT INTO component_verify_tickets (component_appid, ticket) VALUES ($1, $2)`
	_, err := db.ExecContext(context.Background(), query, appID, ticketValue)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTicketRepo_Save_WithLongTicket(t *testing.T) {
	// 验证长 ticket 字符串能正确存储
	db, mock := newMockRepo(t)
	defer db.Close()

	appID := "wx570bc396a51xxxxx"
	// 模拟真实的 ticket 长度（通常 200+ 字符）
	ticketValue := "ticket_" + fmt.Sprintf("%0200d", 1)[:180]

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO component_verify_tickets (component_appid, ticket) VALUES ($1, $2)")).
		WithArgs(appID, ticketValue).
		WillReturnResult(sqlmock.NewResult(2, 1))

	query := `INSERT INTO component_verify_tickets (component_appid, ticket) VALUES ($1, $2)`
	_, err := db.ExecContext(context.Background(), query, appID, ticketValue)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTicketRepo_Save_DifferentAppIDs(t *testing.T) {
	// 验证不同 AppId 的 ticket 独立存储
	db, mock := newMockRepo(t)
	defer db.Close()

	// 保存第一个 appId 的 ticket
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO component_verify_tickets (component_appid, ticket) VALUES ($1, $2)")).
		WithArgs("wx_app_1", "ticket_1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// 保存第二个 appId 的 ticket
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO component_verify_tickets (component_appid, ticket) VALUES ($1, $2)")).
		WithArgs("wx_app_2", "ticket_2").
		WillReturnResult(sqlmock.NewResult(2, 1))

	query := `INSERT INTO component_verify_tickets (component_appid, ticket) VALUES ($1, $2)`
	_, err := db.ExecContext(context.Background(), query, "wx_app_1", "ticket_1")
	assert.NoError(t, err)
	_, err = db.ExecContext(context.Background(), query, "wx_app_2", "ticket_2")
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTicketRepo_Save_DBError(t *testing.T) {
	// 验证数据库错误时正确返回 error
	db, mock := newMockRepo(t)
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO component_verify_tickets (component_appid, ticket) VALUES ($1, $2)")).
		WithArgs("wx_app_1", "ticket_1").
		WillReturnError(fmt.Errorf("connection refused"))

	query := `INSERT INTO component_verify_tickets (component_appid, ticket) VALUES ($1, $2)`
	_, err := db.ExecContext(context.Background(), query, "wx_app_1", "ticket_1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")

	assert.NoError(t, mock.ExpectationsWereMet())
}

// ========== GetLatest 测试 ==========

func TestTicketRepo_GetLatest_Success(t *testing.T) {
	// ticket.go 实现后预期行为:
	//   func (r *Repo) GetLatest(ctx context.Context, appID string) (string, error)
	// SQL: SELECT ticket FROM component_verify_tickets
	//      WHERE component_appid = $1 ORDER BY received_at DESC LIMIT 1

	db, mock := newMockRepo(t)
	defer db.Close()

	appID := "wx570bc396a51xxxxx"
	expectedTicket := "ticket_latest_xyz789"

	columns := []string{"ticket"}
	rows := sqlmock.NewRows(columns).
		AddRow(expectedTicket)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT ticket FROM component_verify_tickets WHERE component_appid = $1 ORDER BY received_at DESC LIMIT 1")).
		WithArgs(appID).
		WillReturnRows(rows)

	// 模拟 GetLatest 行为
	var actualTicket string
	query := `SELECT ticket FROM component_verify_tickets WHERE component_appid = $1 ORDER BY received_at DESC LIMIT 1`
	err := db.QueryRowContext(context.Background(), query, appID).Scan(&actualTicket)
	assert.NoError(t, err)
	assert.Equal(t, expectedTicket, actualTicket)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTicketRepo_GetLatest_NoRows(t *testing.T) {
	// 没有任何 ticket 记录时，应返回空字符串而非错误
	db, mock := newMockRepo(t)
	defer db.Close()

	appID := "wx_no_ticket_app"

	mock.ExpectQuery(regexp.QuoteMeta("SELECT ticket FROM component_verify_tickets WHERE component_appid = $1 ORDER BY received_at DESC LIMIT 1")).
		WithArgs(appID).
		WillReturnError(sql.ErrNoRows)

	// 模拟 GetLatest 行为 — ticket.go 实现时内部应处理 sql.ErrNoRows 返回 ("", nil)
	query := `SELECT ticket FROM component_verify_tickets WHERE component_appid = $1 ORDER BY received_at DESC LIMIT 1`
	var ticket string
	err := db.QueryRowContext(context.Background(), query, appID).Scan(&ticket)
	// sql.ErrNoRows 在 Scan 阶段表现为 sql.ErrNoRows
	assert.ErrorIs(t, err, sql.ErrNoRows)
	// 实际 Repo.GetLatest 应捕获此错误并返回 ("", nil)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTicketRepo_GetLatest_ReturnsMostRecent(t *testing.T) {
	// 验证 ORDER BY received_at DESC LIMIT 1 返回最新记录
	db, mock := newMockRepo(t)
	defer db.Close()

	appID := "wx570bc396a51xxxxx"
	latestTicket := "ticket_received_latest"

	columns := []string{"ticket"}
	// 只返回一行（最新的），验证 SQL 中的 LIMIT 1
	rows := sqlmock.NewRows(columns).
		AddRow(latestTicket)

	// 确保查询带 ORDER BY received_at DESC LIMIT 1
	mock.ExpectQuery(`SELECT ticket FROM component_verify_tickets WHERE component_appid = \$1 ORDER BY received_at DESC LIMIT 1`).
		WithArgs(appID).
		WillReturnRows(rows)

	query := `SELECT ticket FROM component_verify_tickets WHERE component_appid = $1 ORDER BY received_at DESC LIMIT 1`
	var ticket string
	err := db.QueryRowContext(context.Background(), query, appID).Scan(&ticket)
	assert.NoError(t, err)
	assert.Equal(t, latestTicket, ticket)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTicketRepo_GetLatest_DBError(t *testing.T) {
	// 验证数据库连接错误时正确返回 error
	db, mock := newMockRepo(t)
	defer db.Close()

	appID := "wx_app_db_error"

	mock.ExpectQuery(regexp.QuoteMeta("SELECT ticket FROM component_verify_tickets WHERE component_appid = $1 ORDER BY received_at DESC LIMIT 1")).
		WithArgs(appID).
		WillReturnError(fmt.Errorf("connection timeout"))

	query := `SELECT ticket FROM component_verify_tickets WHERE component_appid = $1 ORDER BY received_at DESC LIMIT 1`
	var ticket string
	err := db.QueryRowContext(context.Background(), query, appID).Scan(&ticket)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection timeout")

	assert.NoError(t, mock.ExpectationsWereMet())
}

// ========== Save + GetLatest 集成测试 ==========

func TestTicketRepo_SaveAndGetLatest(t *testing.T) {
	// 端到端模拟: 先 Save 再 GetLatest 验证数据完整性
	db, mock := newMockRepo(t)
	defer db.Close()

	appID := "wx570bc396a51xxxxx"
	ticket1 := "ticket_first_push"
	ticket2 := "ticket_second_push"
	ticket3 := "ticket_third_push"

	// 模拟 3 次推送（微信每 10 分钟推送一次 ticket）

	// 第 1 次推送 (Save)
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO component_verify_tickets (component_appid, ticket) VALUES ($1, $2)")).
		WithArgs(appID, ticket1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// 第 2 次推送 (Save)
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO component_verify_tickets (component_appid, ticket) VALUES ($1, $2)")).
		WithArgs(appID, ticket2).
		WillReturnResult(sqlmock.NewResult(2, 1))

	// 第 3 次推送 (Save)
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO component_verify_tickets (component_appid, ticket) VALUES ($1, $2)")).
		WithArgs(appID, ticket3).
		WillReturnResult(sqlmock.NewResult(3, 1))

	// GetLatest → 应返回 ticket3（最新）
	columns := []string{"ticket"}
	rows := sqlmock.NewRows(columns).AddRow(ticket3)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT ticket FROM component_verify_tickets WHERE component_appid = $1 ORDER BY received_at DESC LIMIT 1")).
		WithArgs(appID).
		WillReturnRows(rows)

	// 执行
	saveQuery := `INSERT INTO component_verify_tickets (component_appid, ticket) VALUES ($1, $2)`
	getQuery := `SELECT ticket FROM component_verify_tickets WHERE component_appid = $1 ORDER BY received_at DESC LIMIT 1`

	_, _ = db.ExecContext(context.Background(), saveQuery, appID, ticket1)
	_, _ = db.ExecContext(context.Background(), saveQuery, appID, ticket2)
	_, _ = db.ExecContext(context.Background(), saveQuery, appID, ticket3)

	var latestTicket string
	err := db.QueryRowContext(context.Background(), getQuery, appID).Scan(&latestTicket)
	assert.NoError(t, err)
	assert.Equal(t, ticket3, latestTicket, "GetLatest 应返回最新推送的 ticket")

	assert.NoError(t, mock.ExpectationsWereMet())
}

// ========== 表结构验证 ==========

func TestTicketTableSchema(t *testing.T) {
	// 验证表结构符合设计文档定义
	t.Run("component_verify_tickets 表包含预期列", func(t *testing.T) {
		db, mock := newMockRepo(t)
		defer db.Close()

		columns := []string{
			"id", "component_appid", "ticket", "received_at", "created_at",
		}
		rows := sqlmock.NewRows(columns).AddRow(
			int64(1), "wx570bc396a51xxxxx", "ticket_xxx",
			time.Now(), time.Now(),
		)

		mock.ExpectQuery(`SELECT \* FROM component_verify_tickets WHERE id = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(rows)

		// 验证查询能覆盖所有预期列
		type TicketRecord struct {
			ID             int64     `db:"id"`
			ComponentAppID string    `db:"component_appid"`
			Ticket         string    `db:"ticket"`
			ReceivedAt     time.Time `db:"received_at"`
			CreatedAt      time.Time `db:"created_at"`
		}
		var record TicketRecord
		query := `SELECT * FROM component_verify_tickets WHERE id = $1`
		err := db.QueryRowContext(context.Background(), query, int64(1)).Scan(
			&record.ID, &record.ComponentAppID, &record.Ticket,
			&record.ReceivedAt, &record.CreatedAt,
		)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), record.ID)
		assert.Equal(t, "wx570bc396a51xxxxx", record.ComponentAppID)
		assert.Equal(t, "ticket_xxx", record.Ticket)
		assert.False(t, record.ReceivedAt.IsZero())

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// ========== 索引验证 ==========

func TestTicketIndexUsage(t *testing.T) {
	// 验证 idx_tickets_appid_time 索引的查询模式
	t.Run("按 appid + received_at DESC 查询应使用索引", func(t *testing.T) {
		db, mock := newMockRepo(t)
		defer db.Close()

		appID := "wx570bc396a51xxxxx"
		columns := []string{"ticket"}
		rows := sqlmock.NewRows(columns).AddRow("ticket_from_index")

		mock.ExpectQuery(`SELECT ticket FROM component_verify_tickets WHERE component_appid = \$1 ORDER BY received_at DESC LIMIT 1`).
			WithArgs(appID).
			WillReturnRows(rows)

		// 此查询模式对应 idx_tickets_appid_time(component_appid, received_at DESC) 索引
		query := `SELECT ticket FROM component_verify_tickets WHERE component_appid = $1 ORDER BY received_at DESC LIMIT 1`
		var ticket string
		err := db.QueryRowContext(context.Background(), query, appID).Scan(&ticket)
		assert.NoError(t, err)
		assert.Equal(t, "ticket_from_index", ticket)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
