package cache

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/mohammadraufzahed/translate-mcp/internal/config"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

type sqlCache struct {
	db      *sql.DB
	dialect string
}

func newSQLCache(cfg config.CacheTierConfig) (Cache, error) {
	dialect := cfg.Type
	dsn := cfg.DSN
	if dsn == "" {
		if dialect == "sqlite" {
			dsn = "translations.db?_journal=WAL&_busy_timeout=5000"
		} else {
			return nil, fmt.Errorf("postgres dsn required")
		}
	}
	var driver string
	if dialect == "sqlite" {
		driver = "sqlite"
		if !strings.Contains(dsn, "_journal") && !strings.Contains(dsn, ":memory:") {
			dsn = dsn + "?_journal=WAL&_busy_timeout=5000"
		}
	} else {
		driver = "pgx"
	}
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxLifetime(time.Hour)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(2)

	s := &sqlCache{db: db, dialect: dialect}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *sqlCache) migrate() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var textType string
	if s.dialect == "sqlite" {
		textType = "TEXT"
	} else {
		textType = "TEXT"
	}
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS translations (
  key_hash %s PRIMARY KEY,
  response_json %s NOT NULL,
  created_at TIMESTAMP NOT NULL
)`, textType, textType))
	return err
}

func (s *sqlCache) Get(ctx context.Context, key string) (*Item, bool, error) {
	var data []byte
	var created time.Time
	err := s.db.QueryRowContext(ctx, s.selectSQL("SELECT response_json, created_at FROM translations WHERE key_hash ="), key).Scan(&data, &created)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	item, err := DeserializeItem(data)
	if err != nil {
		return nil, false, err
	}
	item.CreatedAt = created
	return item, true, nil
}

func (s *sqlCache) Set(ctx context.Context, key string, item *Item, ttl time.Duration) error {
	data, err := SerializeItem(item)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, s.upsertSQL(), key, data, item.CreatedAt)
	return err
}

func (s *sqlCache) Close() error {
	return s.db.Close()
}

func (s *sqlCache) selectSQL(base string) string {
	if s.dialect == "postgres" {
		return base + " $1"
	}
	return base + " ?"
}

func (s *sqlCache) upsertSQL() string {
	if s.dialect == "postgres" {
		return `INSERT INTO translations (key_hash, response_json, created_at) VALUES ($1, $2, $3)
				ON CONFLICT (key_hash) DO UPDATE SET response_json = EXCLUDED.response_json, created_at = EXCLUDED.created_at`
	}
	return `INSERT INTO translations (key_hash, response_json, created_at) VALUES (?, ?, ?)
			ON CONFLICT (key_hash) DO UPDATE SET response_json = excluded.response_json, created_at = excluded.created_at`
}
