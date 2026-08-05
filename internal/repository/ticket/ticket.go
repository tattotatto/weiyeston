// Package ticket 微信 component_verify_ticket 持久化 Repository
package ticket

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
)

// Repo component_verify_ticket 数据访问层
type Repo struct {
	DB *sqlx.DB
}

// Save 保存 component_verify_ticket 到数据库
func (r *Repo) Save(ctx context.Context, appID string, ticket string) error {
	query := `INSERT INTO component_verify_tickets (component_appid, ticket) VALUES ($1, $2)`
	_, err := r.DB.ExecContext(ctx, query, appID, ticket)
	return err
}

// GetLatest 获取最近一条 component_verify_ticket
// 没有记录时返回空字符串和 nil error
func (r *Repo) GetLatest(ctx context.Context, appID string) (string, error) {
	var record struct {
		Ticket string `db:"ticket"`
	}
	query := `SELECT ticket FROM component_verify_tickets WHERE component_appid = $1 ORDER BY received_at DESC LIMIT 1`
	err := r.DB.GetContext(ctx, &record, query, appID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return record.Ticket, nil
}
