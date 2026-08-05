// migrate — standalone CLI tool that migrates data from a legacy SQL Server
// database to the new PostgreSQL database used by weiyeston-v2.
//
// Usage:
//
//	go run ./tools/migrate --config=config.yaml --source-db="sqlserver://sa:password@localhost:1433?database=WeiYesTon"
//
// The tool runs migrations in a fixed order (tenants, accounts, channels, articles,
// quicknews, auto-reply) with idempotent UPSERTs so it can be re-run safely.
package main

import (
	"database/sql"
	ej "encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	_ "github.com/microsoft/go-mssqldb"
	"github.com/weiyeston/weiyeston-v2/internal/config"
	"golang.org/x/crypto/bcrypt"
	htmlpkg "golang.org/x/net/html"
)

// migration token that links rows across the legacy tables.
const defaultToken = "qWubTV85033743"

// target tenant id in the new database.
const tenantID = 25

func main() {
	configPath := flag.String("config", "config.yaml", "Path to YAML config file (PostgreSQL settings)")
	sourceDSN := flag.String("source-db", "", "SQL Server DSN (e.g. sqlserver://sa:admin4wwj@localhost:1433?database=WeiYesTon)")
	flag.Parse()

	if *sourceDSN == "" {
		fmt.Fprintf(os.Stderr, "Error: --source-db is required\n")
		flag.Usage()
		os.Exit(1)
	}

	// ── Load PostgreSQL config ──────────────────────────────────────────
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config from %q: %v", *configPath, err)
	}

	pgDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User,
		cfg.Database.Password, cfg.Database.DBName, cfg.Database.SSLMode)

	log.Printf("PostgreSQL target: %s:%d/%s", cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)
	log.Printf("SQL Server source: %s", *sourceDSN)

	// ── Open connections ────────────────────────────────────────────────
	srcDB, err := sql.Open("sqlserver", *sourceDSN)
	if err != nil {
		log.Fatalf("Failed to open SQL Server: %v", err)
	}
	defer srcDB.Close()
	if err := srcDB.Ping(); err != nil {
		log.Fatalf("Failed to ping SQL Server: %v", err)
	}
	log.Println("Connected to SQL Server")

	tgtDB, err := sqlx.Connect("postgres", pgDSN)
	if err != nil {
		log.Fatalf("Failed to connect PostgreSQL: %v", err)
	}
	defer tgtDB.Close()
	log.Println("Connected to PostgreSQL")

	// ── Run migrations in order ─────────────────────────────────────────
	steps := []struct {
		name string
		fn   func(*sql.DB, *sqlx.DB) error
	}{
		{"1. WYT_ADMIN_USERS → tenants", migrateTenant},
		{"2. WYT_USER_MP → wechat_accounts", migrateAccount},
		{"3. WYT_WEB_CHNSet → cms_channels", migrateChannels},
		{"4. WYT_WEB_CONTENTSSet → cms_articles", migrateArticles},
		{"5. WYT_QUICKNEWS_NEWSSet → quicknews_news", migrateQuickNews},
		{"6. WYT_QUICKNEWS_CHNSet → quicknews_channels", migrateQuickNewsChannels},
		{"7. WYT_MP_KEYWORDS → auto_reply_rules", migrateKeywords},
		{"8. WYT_MP_DEFAULT → auto_reply_rules (default)", migrateDefaultReply},
	}

	for _, s := range steps {
		log.Printf("--- %s ---", s.name)
		if err := s.fn(srcDB, tgtDB); err != nil {
			log.Fatalf("Migration step %q failed: %v", s.name, err)
		}
		log.Printf("✓ %s completed", s.name)
	}

	log.Println("========================================")
	log.Println("Migration completed successfully.")
}

// ── Helpers: read SQL Server rows into map slices ──────────────────────────

func queryRows(db *sql.DB, query string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	lowerCols := make([]string, len(cols))
	for i, c := range cols {
		lowerCols[i] = strings.ToLower(c)
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		row := make(map[string]interface{})
		for i, col := range lowerCols {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}
	return results, rows.Err()
}

func queryRow(db *sql.DB, query string, args ...interface{}) (map[string]interface{}, error) {
	results, err := queryRows(db, query, args...)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, sql.ErrNoRows
	}
	return results[0], nil
}

// ── Helpers: typed accessors for map[string]interface{} ────────────────────

func getString(row map[string]interface{}, key string) string {
	v, ok := row[strings.ToLower(key)]
	if !ok || v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case time.Time:
		return val.Format("2006-01-02 15:04:05")
	case bool:
		if val {
			return "1"
		}
		return "0"
	default:
		return fmt.Sprintf("%v", val)
	}
}

func getStringPtr(row map[string]interface{}, key string) *string {
	s := getString(row, key)
	if s == "" {
		return nil
	}
	return &s
}

func getInt(row map[string]interface{}, key string) int {
	v, ok := row[strings.ToLower(key)]
	if !ok || v == nil {
		return 0
	}
	switch val := v.(type) {
	case int64:
		return int(val)
	case float64:
		return int(val)
	case string:
		i, _ := strconv.Atoi(val)
		return i
	default:
		return 0
	}
}

func getInt16(row map[string]interface{}, key string) int16 {
	return int16(getInt(row, key))
}

func getInt64(row map[string]interface{}, key string) int64 {
	v, ok := row[strings.ToLower(key)]
	if !ok || v == nil {
		return 0
	}
	switch val := v.(type) {
	case int64:
		return val
	case float64:
		return int64(val)
	case string:
		i, _ := strconv.ParseInt(val, 10, 64)
		return i
	default:
		return 0
	}
}

func getBool(row map[string]interface{}, key string) bool {
	v, ok := row[strings.ToLower(key)]
	if !ok || v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case int64:
		return val != 0
	case float64:
		return val != 0
	case string:
		return val == "1" || strings.EqualFold(val, "true")
	default:
		return false
	}
}

func getTime(row map[string]interface{}, key string) time.Time {
	v, ok := row[strings.ToLower(key)]
	if !ok || v == nil {
		return time.Now()
	}
	if t, ok := v.(time.Time); ok {
		return t
	}
	return time.Now()
}

func getTimePtr(row map[string]interface{}, key string) *time.Time {
	v, ok := row[strings.ToLower(key)]
	if !ok || v == nil {
		return nil
	}
	if t, ok := v.(time.Time); ok {
		return &t
	}
	return nil
}

// strPtr returns the string pointer's value or "<nil>" for logging.
func strPtr(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}

// ── Step 1: WYT_ADMIN_USERS → tenants ─────────────────────────────────────

func migrateTenant(src *sql.DB, tgt *sqlx.DB) error {
	row, err := queryRow(src, "SELECT * FROM WYT_ADMIN_USERS WHERE id=?", 25)
	if err == sql.ErrNoRows {
		log.Println("  No WYT_ADMIN_USERS row with id=25 — skipping")
		return nil
	}
	if err != nil {
		return fmt.Errorf("query WYT_ADMIN_USERS: %w", err)
	}

	username := getString(row, "username")
	passwordMD5 := getString(row, "userpwd")
	nickname := getStringPtr(row, "nickname")
	email := getStringPtr(row, "email")
	phone := getStringPtr(row, "phone")
	role := getString(row, "role")
	if role == "" {
		role = "admin"
	}
	status := getInt16(row, "status")
	if status == 0 {
		status = 1
	}
	lastLogin := getTimePtr(row, "last_login")
	createdAt := getTime(row, "created")
	updatedAt := getTime(row, "updated")
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}

	// Convert MD5 hash to bcrypt.
	bcryptHash, err := bcrypt.GenerateFromPassword([]byte(passwordMD5), 12)
	if err != nil {
		return fmt.Errorf("bcrypt hash for tenant %q: %w", username, err)
	}

	_, err = tgt.Exec(`
		INSERT INTO tenants (id, username, password_hash, nickname, email, phone, role, status, last_login_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			username = EXCLUDED.username,
			password_hash = EXCLUDED.password_hash,
			nickname = EXCLUDED.nickname,
			email = EXCLUDED.email,
			phone = EXCLUDED.phone,
			role = EXCLUDED.role,
			status = EXCLUDED.status,
			last_login_at = EXCLUDED.last_login_at,
			updated_at = EXCLUDED.updated_at
	`, tenantID, username, string(bcryptHash), nickname, email, phone, role, status,
		lastLogin, createdAt, updatedAt)
	if err != nil {
		return fmt.Errorf("insert tenant: %w", err)
	}

	log.Printf("  Migrated tenant: id=%d username=%q", tenantID, username)
	return nil
}

// ── Step 2: WYT_USER_MP → wechat_accounts ─────────────────────────────────

func migrateAccount(src *sql.DB, tgt *sqlx.DB) error {
	row, err := queryRow(src, "SELECT * FROM WYT_USER_MP WHERE token=?", defaultToken)
	if err == sql.ErrNoRows {
		log.Printf("  No WYT_USER_MP row with token=%q — skipping", defaultToken)
		return nil
	}
	if err != nil {
		return fmt.Errorf("query WYT_USER_MP: %w", err)
	}

	id := getInt64(row, "id")
	name := getStringPtr(row, "name")
	wxOriginalID := getStringPtr(row, "wx_original_id")
	wxAppID := getStringPtr(row, "wx_app_id")
	wxAppSecret := getStringPtr(row, "wx_app_secret")
	description := getStringPtr(row, "description")
	avatarURL := getStringPtr(row, "avatar_url")
	qrcodeURL := getStringPtr(row, "qr_code_url")
	status := getInt16(row, "status")
	if status == 0 {
		status = 1
	}
	createdAt := getTime(row, "created")
	updatedAt := getTime(row, "updated")
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}

	authType := int16(1) // manual

	_, err = tgt.Exec(`
		INSERT INTO wechat_accounts (id, tenant_id, name, wx_original_id, wx_app_id, wx_app_secret,
			auth_type, avatar_url, qr_code_url, description, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (id) DO UPDATE SET
			tenant_id = EXCLUDED.tenant_id,
			name = EXCLUDED.name,
			wx_original_id = EXCLUDED.wx_original_id,
			wx_app_id = EXCLUDED.wx_app_id,
			wx_app_secret = EXCLUDED.wx_app_secret,
			auth_type = EXCLUDED.auth_type,
			avatar_url = EXCLUDED.avatar_url,
			qr_code_url = EXCLUDED.qr_code_url,
			description = EXCLUDED.description,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at
	`, id, tenantID, name, wxOriginalID, wxAppID, wxAppSecret,
		authType, avatarURL, qrcodeURL, description, status, createdAt, updatedAt)
	if err != nil {
		return fmt.Errorf("insert wechat_account: %w", err)
	}

	log.Printf("  Migrated account: id=%d name=%q", id, strPtr(name))
	return nil
}

// ── Step 3: WYT_WEB_CHNSet → cms_channels ─────────────────────────────────

func migrateChannels(src *sql.DB, tgt *sqlx.DB) error {
	rows, err := queryRows(src, "SELECT * FROM WYT_WEB_CHNSet WHERE token=?", defaultToken)
	if err != nil {
		return fmt.Errorf("query WYT_WEB_CHNSet: %w", err)
	}

	count := 0
	for _, row := range rows {
		id := getInt64(row, "id")
		parentID := (*int64)(nil)
		if pid := getInt(row, "parentid"); pid > 0 {
			p := int64(pid)
			parentID = &p
		}
		name := getString(row, "name")
		level := getInt16(row, "level")
		sortOrder := getInt(row, "sortorder")
		coverURL := getStringPtr(row, "cover_url")
		description := getStringPtr(row, "description")
		status := getInt16(row, "status")
		if status == 0 {
			status = 1
		}
		createdAt := getTime(row, "created")
		updatedAt := getTime(row, "updated")
		if updatedAt.IsZero() {
			updatedAt = createdAt
		}

		_, err := tgt.Exec(`
			INSERT INTO cms_channels (id, account_id, parent_id, name, level, sort_order, cover_url, description, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			ON CONFLICT (id) DO UPDATE SET
				account_id = EXCLUDED.account_id,
				parent_id = EXCLUDED.parent_id,
				name = EXCLUDED.name,
				level = EXCLUDED.level,
				sort_order = EXCLUDED.sort_order,
				cover_url = EXCLUDED.cover_url,
				description = EXCLUDED.description,
				status = EXCLUDED.status,
				updated_at = EXCLUDED.updated_at
		`, id, tenantID, parentID, name, level, sortOrder, coverURL, description, status, createdAt, updatedAt)
		if err != nil {
			return fmt.Errorf("insert cms_channel id=%d: %w", id, err)
		}
		count++
	}

	log.Printf("  Migrated %d channels", count)
	return nil
}

// ── Step 4: WYT_WEB_CONTENTSSet → cms_articles ────────────────────────────

func migrateArticles(src *sql.DB, tgt *sqlx.DB) error {
	rows, err := queryRows(src, "SELECT * FROM WYT_WEB_CONTENTSSet WHERE token=?", defaultToken)
	if err != nil {
		return fmt.Errorf("query WYT_WEB_CONTENTSSet: %w", err)
	}

	count := 0
	for _, row := range rows {
		id := getInt64(row, "id")
		channelID := (*int64)(nil)
		if cid := getInt(row, "chn_id"); cid > 0 {
			c := int64(cid)
			channelID = &c
		}
		title := getStringPtr(row, "title")
		coverURL := getStringPtr(row, "cover_url")
		summary := getStringPtr(row, "summary")
		author := getStringPtr(row, "author")
		htmlContent := getString(row, "content")
		tipTapJSON := htmlToTipTapJSON(htmlContent)
		status := getInt16(row, "status")
		if status == 0 {
			status = 1
		}
		sortOrder := getInt(row, "sort_order")
		viewCount := getInt(row, "view_count")
		publishedAt := getTimePtr(row, "published_at")
		createdAt := getTime(row, "created")
		updatedAt := getTime(row, "updated")
		if updatedAt.IsZero() {
			updatedAt = createdAt
		}

		_, err := tgt.Exec(`
			INSERT INTO cms_articles (id, account_id, channel_id, title, cover_url, summary, author, content, status, sort_order, view_count, published_at, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
			ON CONFLICT (id) DO UPDATE SET
				account_id = EXCLUDED.account_id,
				channel_id = EXCLUDED.channel_id,
				title = EXCLUDED.title,
				cover_url = EXCLUDED.cover_url,
				summary = EXCLUDED.summary,
				author = EXCLUDED.author,
				content = EXCLUDED.content,
				status = EXCLUDED.status,
				sort_order = EXCLUDED.sort_order,
				view_count = EXCLUDED.view_count,
				published_at = EXCLUDED.published_at,
				updated_at = EXCLUDED.updated_at
		`, id, tenantID, channelID, title, coverURL, summary, author, tipTapJSON, status, sortOrder, viewCount, publishedAt, createdAt, updatedAt)
		if err != nil {
			return fmt.Errorf("insert cms_article id=%d: %w", id, err)
		}
		count++
	}

	log.Printf("  Migrated %d articles", count)
	return nil
}

// ── Step 5: WYT_QUICKNEWS_NEWSSet → quicknews_news ────────────────────────

func migrateQuickNews(src *sql.DB, tgt *sqlx.DB) error {
	rows, err := queryRows(src, "SELECT * FROM WYT_QUICKNEWS_NEWSSet WHERE token=?", defaultToken)
	if err != nil {
		return fmt.Errorf("query WYT_QUICKNEWS_NEWSSet: %w", err)
	}

	count := 0
	for _, row := range rows {
		id := getInt64(row, "id")
		channelID := getInt64(row, "chn_id")
		content := getString(row, "content")
		authorName := getStringPtr(row, "author_name")
		authorAvatar := getStringPtr(row, "author_avatar")
		status := getInt16(row, "status")
		if status == 0 {
			status = 1
		}
		likeCount := getInt(row, "like_count")
		commentCount := getInt(row, "comment_count")
		isTop := getBool(row, "is_top")
		createdAt := getTime(row, "created")
		updatedAt := getTime(row, "updated")
		if updatedAt.IsZero() {
			updatedAt = createdAt
		}

		_, err := tgt.Exec(`
			INSERT INTO quicknews_news (id, account_id, channel_id, content, author_name, author_avatar, status, like_count, comment_count, is_top, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			ON CONFLICT (id) DO UPDATE SET
				account_id = EXCLUDED.account_id,
				channel_id = EXCLUDED.channel_id,
				content = EXCLUDED.content,
				author_name = EXCLUDED.author_name,
				author_avatar = EXCLUDED.author_avatar,
				status = EXCLUDED.status,
				like_count = EXCLUDED.like_count,
				comment_count = EXCLUDED.comment_count,
				is_top = EXCLUDED.is_top,
				updated_at = EXCLUDED.updated_at
		`, id, tenantID, channelID, content, authorName, authorAvatar, status, likeCount, commentCount, isTop, createdAt, updatedAt)
		if err != nil {
			return fmt.Errorf("insert quicknews_news id=%d: %w", id, err)
		}
		count++
	}

	log.Printf("  Migrated %d quicknews items", count)
	return nil
}

// ── Step 6: WYT_QUICKNEWS_CHNSet → quicknews_channels ─────────────────────

func migrateQuickNewsChannels(src *sql.DB, tgt *sqlx.DB) error {
	rows, err := queryRows(src, "SELECT * FROM WYT_QUICKNEWS_CHNSet WHERE token=?", defaultToken)
	if err != nil {
		return fmt.Errorf("query WYT_QUICKNEWS_CHNSet: %w", err)
	}

	count := 0
	for _, row := range rows {
		id := getInt64(row, "id")
		name := getString(row, "name")
		coverURL := getStringPtr(row, "cover_url")
		description := getStringPtr(row, "description")
		sortOrder := getInt(row, "sort_order")
		status := getInt16(row, "status")
		if status == 0 {
			status = 1
		}
		createdAt := getTime(row, "created")
		updatedAt := getTime(row, "updated")
		if updatedAt.IsZero() {
			updatedAt = createdAt
		}

		_, err := tgt.Exec(`
			INSERT INTO quicknews_channels (id, account_id, name, cover_url, description, sort_order, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (id) DO UPDATE SET
				account_id = EXCLUDED.account_id,
				name = EXCLUDED.name,
				cover_url = EXCLUDED.cover_url,
				description = EXCLUDED.description,
				sort_order = EXCLUDED.sort_order,
				status = EXCLUDED.status,
				updated_at = EXCLUDED.updated_at
		`, id, tenantID, name, coverURL, description, sortOrder, status, createdAt, updatedAt)
		if err != nil {
			return fmt.Errorf("insert quicknews_channel id=%d: %w", id, err)
		}
		count++
	}

	log.Printf("  Migrated %d quicknews channels", count)
	return nil
}

// ── Step 7: WYT_MP_KEYWORDS → auto_reply_rules ────────────────────────────

func migrateKeywords(src *sql.DB, tgt *sqlx.DB) error {
	rows, err := queryRows(src, "SELECT * FROM WYT_MP_KEYWORDS WHERE token=?", defaultToken)
	if err != nil {
		return fmt.Errorf("query WYT_MP_KEYWORDS: %w", err)
	}

	count := 0
	for _, row := range rows {
		id := getInt64(row, "id")
		keyword := getStringPtr(row, "keyword")
		matchType := getInt16(row, "match_type")
		replyType := getInt16(row, "reply_type")
		replyContent := getString(row, "reply_content")
		status := getInt16(row, "status")
		if status == 0 {
			status = 1
		}
		createdAt := getTime(row, "created")
		updatedAt := getTime(row, "updated")
		if updatedAt.IsZero() {
			updatedAt = createdAt
		}

		_, err := tgt.Exec(`
			INSERT INTO auto_reply_rules (id, account_id, keyword, match_type, reply_type, reply_content, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (id) DO UPDATE SET
				account_id = EXCLUDED.account_id,
				keyword = EXCLUDED.keyword,
				match_type = EXCLUDED.match_type,
				reply_type = EXCLUDED.reply_type,
				reply_content = EXCLUDED.reply_content,
				status = EXCLUDED.status,
				updated_at = EXCLUDED.updated_at
		`, id, tenantID, keyword, matchType, replyType, replyContent, status, createdAt, updatedAt)
		if err != nil {
			return fmt.Errorf("insert auto_reply_rule (keyword) id=%d: %w", id, err)
		}
		count++
	}

	log.Printf("  Migrated %d keyword reply rules", count)
	return nil
}

// ── Step 8: WYT_MP_DEFAULT → auto_reply_rules (default reply) ─────────────

func migrateDefaultReply(src *sql.DB, tgt *sqlx.DB) error {
	rows, err := queryRows(src, "SELECT * FROM WYT_MP_DEFAULT WHERE token=?", defaultToken)
	if err != nil {
		return fmt.Errorf("query WYT_MP_DEFAULT: %w", err)
	}

	count := 0
	for _, row := range rows {
		id := getInt64(row, "id")
		// Default reply: no keyword, exact match type (0).
		var keyword *string = nil
		matchType := int16(0)
		replyType := getInt16(row, "reply_type")
		replyContent := getString(row, "content")
		if replyContent == "" {
			replyContent = getString(row, "reply_content")
		}
		status := getInt16(row, "status")
		if status == 0 {
			status = 1
		}
		createdAt := getTime(row, "created")
		updatedAt := getTime(row, "updated")
		if updatedAt.IsZero() {
			updatedAt = createdAt
		}

		_, err := tgt.Exec(`
			INSERT INTO auto_reply_rules (id, account_id, keyword, match_type, reply_type, reply_content, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (id) DO UPDATE SET
				account_id = EXCLUDED.account_id,
				keyword = EXCLUDED.keyword,
				match_type = EXCLUDED.match_type,
				reply_type = EXCLUDED.reply_type,
				reply_content = EXCLUDED.reply_content,
				status = EXCLUDED.status,
				updated_at = EXCLUDED.updated_at
		`, id, tenantID, keyword, matchType, replyType, replyContent, status, createdAt, updatedAt)
		if err != nil {
			return fmt.Errorf("insert auto_reply_rule (default) id=%d: %w", id, err)
		}
		count++
	}

	log.Printf("  Migrated %d default reply rules", count)
	return nil
}

// ── HTML → TipTap JSONB converter ─────────────────────────────────────────

// htmlToTipTapJSON converts an HTML string into a TipTap-compatible JSON array.
// The result is a JSON array string, e.g. [{"type":"paragraph","content":[...]}].
// Empty or whitespace-only input returns "[]".
func htmlToTipTapJSON(htmlContent string) string {
	htmlContent = strings.TrimSpace(htmlContent)
	if htmlContent == "" {
		return "[]"
	}

	// Parse the HTML fragment. Wrap in a context element so the parser
	// treats the content as body-level children.
	ctx := &htmlpkg.Node{Type: htmlpkg.ElementNode, Data: "body"}
	nodes, err := htmlpkg.ParseFragment(strings.NewReader(htmlContent), ctx)
	if err != nil {
		log.Printf("Warning: failed to parse HTML fragment: %v; storing raw", err)
		// Fallback: wrap raw HTML in a text block.
		raw := fmt.Sprintf(`[{"type":"paragraph","content":[{"type":"text","text":%q}]}]`, htmlContent)
		return raw
	}

	var result []map[string]interface{}
	for _, node := range nodes {
		convertBlock(node, &result)
	}

	b, err := ej.Marshal(result)
	if err != nil {
		log.Printf("Warning: failed to marshal TipTap JSON: %v", err)
		return "[]"
	}
	return string(b)
}

// convertBlock converts a top-level HTML node into a TipTap block node
// and appends it to result.
func convertBlock(node *htmlpkg.Node, result *[]map[string]interface{}) {
	if node.Type == htmlpkg.TextNode {
		text := strings.TrimSpace(node.Data)
		if text == "" {
			return
		}
		*result = append(*result, map[string]interface{}{
			"type":    "paragraph",
			"content": []interface{}{map[string]interface{}{"type": "text", "text": text}},
		})
		return
	}

	if node.Type != htmlpkg.ElementNode {
		// Skip comment nodes, doctype, etc. — recurse into children.
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			convertBlock(c, result)
		}
		return
	}

	switch node.Data {
	case "p", "div":
		*result = append(*result, map[string]interface{}{
			"type":    "paragraph",
			"content": collectInlineContent(node),
		})

	case "h1", "h2", "h3", "h4", "h5", "h6":
		level := int(node.Data[1] - '0')
		*result = append(*result, map[string]interface{}{
			"type": "heading",
			"attrs": map[string]interface{}{
				"level": level,
			},
			"content": collectInlineContent(node),
		})

	case "img":
		src := getAttr(node, "src")
		alt := getAttr(node, "alt")
		item := map[string]interface{}{
			"type": "imageBlock",
			"attrs": map[string]interface{}{
				"src": src,
			},
		}
		if alt != "" {
			item["attrs"].(map[string]interface{})["alt"] = alt
		}
		*result = append(*result, item)

	case "ul":
		*result = append(*result, map[string]interface{}{
			"type":    "bulletList",
			"content": collectListItems(node),
		})

	case "ol":
		*result = append(*result, map[string]interface{}{
			"type":    "orderedList",
			"attrs":   map[string]interface{}{},
			"content": collectListItems(node),
		})

	case "blockquote":
		var content []interface{}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == htmlpkg.ElementNode && c.Data == "p" {
				content = append(content, map[string]interface{}{
					"type":    "paragraph",
					"content": collectInlineContent(c),
				})
			} else if c.Type == htmlpkg.TextNode {
				text := strings.TrimSpace(c.Data)
				if text != "" {
					content = append(content, map[string]interface{}{
						"type":    "paragraph",
						"content": []interface{}{map[string]interface{}{"type": "text", "text": text}},
					})
				}
			}
		}
		if len(content) > 0 {
			*result = append(*result, map[string]interface{}{
				"type":    "blockquote",
				"content": content,
			})
		}

	case "pre":
		text := extractText(node)
		*result = append(*result, map[string]interface{}{
			"type": "codeBlock",
			"content": []interface{}{
				map[string]interface{}{"type": "text", "text": text},
			},
		})

	case "br", "hr":
		// Self-closing elements → empty paragraph.
		*result = append(*result, map[string]interface{}{
			"type":    "paragraph",
			"content": []interface{}{},
		})

	default:
		// Unknown block element — recurse into children.
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			convertBlock(c, result)
		}
	}
}

// collectInlineContent walks the children of a block-level element and
// returns a slice of TipTap inline nodes (text nodes with optional marks).
func collectInlineContent(node *htmlpkg.Node) []interface{} {
	var content []interface{}
	collectTextFromNode(node, &content, nil)
	// If there's no content, return a single empty text node so TipTap doesn't error.
	if len(content) == 0 {
		content = append(content, map[string]interface{}{"type": "text", "text": ""})
	}
	return content
}

// collectTextFromNode recursively walks a node tree and builds inline content.
// marks carries the ancestor marks for nested inline formatting (bold, italic, links, etc.).
func collectTextFromNode(node *htmlpkg.Node, content *[]interface{}, marks []map[string]interface{}) {
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		switch c.Type {
		case htmlpkg.TextNode:
			text := c.Data
			// Skip leading whitespace unless it's all we have.
			if len(*content) == 0 && strings.TrimSpace(text) == "" {
				continue
			}
			textNode := map[string]interface{}{
				"type": "text",
				"text": text,
			}
			if len(marks) > 0 {
				// Deep-copy marks so later siblings don't mutate them.
				marksCopy := make([]map[string]interface{}, len(marks))
				for i, m := range marks {
					marksCopy[i] = copyMark(m)
				}
				textNode["marks"] = marksCopy
			}
			*content = append(*content, textNode)

		case htmlpkg.ElementNode:
			switch c.Data {
			case "strong", "b":
				newMarks := appendMark(marks, map[string]interface{}{"type": "bold"})
				collectTextFromNode(c, content, newMarks)

			case "em", "i":
				newMarks := appendMark(marks, map[string]interface{}{"type": "italic"})
				collectTextFromNode(c, content, newMarks)

			case "u":
				newMarks := appendMark(marks, map[string]interface{}{"type": "underline"})
				collectTextFromNode(c, content, newMarks)

			case "s", "strike", "del":
				newMarks := appendMark(marks, map[string]interface{}{"type": "strike"})
				collectTextFromNode(c, content, newMarks)

			case "code":
				newMarks := appendMark(marks, map[string]interface{}{"type": "code"})
				collectTextFromNode(c, content, newMarks)

			case "a":
				href := getAttr(c, "href")
				target := getAttr(c, "target")
				linkMark := map[string]interface{}{"type": "link"}
				attrs := map[string]interface{}{"href": href}
				if target != "" {
					attrs["target"] = target
				}
				linkMark["attrs"] = attrs
				newMarks := appendMark(marks, linkMark)
				collectTextFromNode(c, content, newMarks)

			case "span":
				// Pass through span without adding marks (CSS styling is not preserved).
				collectTextFromNode(c, content, marks)

			case "br":
				// Line break → hard break node.
				*content = append(*content, map[string]interface{}{
					"type": "hardBreak",
				})

			case "img":
				// Inline image — rare but possible.
				src := getAttr(c, "src")
				if src != "" {
					*content = append(*content, map[string]interface{}{
						"type":  "imageInline",
						"attrs": map[string]interface{}{"src": src},
					})
				}

			default:
				// Unknown inline element — recurse without adding marks.
				collectTextFromNode(c, content, marks)
			}
		}
	}
}

// appendMark returns a new marks slice with the given mark appended.
func appendMark(marks []map[string]interface{}, mark map[string]interface{}) []map[string]interface{} {
	out := make([]map[string]interface{}, len(marks)+1)
	copy(out, marks)
	out[len(marks)] = mark
	return out
}

// copyMark returns a shallow copy of the mark map.
func copyMark(m map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// collectListItems extracts <li> children from a <ul> or <ol> node.
func collectListItems(node *htmlpkg.Node) []interface{} {
	var items []interface{}
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != htmlpkg.ElementNode || c.Data != "li" {
			continue
		}
		// Build paragraph content for the list item.
		liContent := []interface{}{
			map[string]interface{}{
				"type":    "paragraph",
				"content": collectInlineContent(c),
			},
		}
		// Check for nested lists inside the <li>.
		for sub := c.FirstChild; sub != nil; sub = sub.NextSibling {
			if sub.Type == htmlpkg.ElementNode && (sub.Data == "ul" || sub.Data == "ol") {
				nestedList := map[string]interface{}{
					"type":    "bulletList",
					"content": collectListItems(sub),
				}
				if sub.Data == "ol" {
					nestedList["type"] = "orderedList"
				}
				// Replace liContent with expanded version including nested list.
				liContent = append(liContent, nestedList)
			}
		}
		items = append(items, map[string]interface{}{
			"type":    "listItem",
			"content": liContent,
		})
	}
	return items
}

// getAttr returns the value of an attribute on an HTML node, or "".
func getAttr(node *htmlpkg.Node, key string) string {
	for _, attr := range node.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

// extractText recursively extracts all text from a node tree.
func extractText(node *htmlpkg.Node) string {
	var buf strings.Builder
	var walk func(*htmlpkg.Node)
	walk = func(n *htmlpkg.Node) {
		if n.Type == htmlpkg.TextNode {
			buf.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(node)
	return buf.String()
}
