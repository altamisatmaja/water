package graph

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sync"

	_ "github.com/marcboeker/go-duckdb"
)

type Client struct {
	db *sql.DB
	mu sync.Mutex // DuckDB single-writer
}

func NewClient(ctx context.Context, dbPath string) (*Client, error) {
	dbFile := filepath.Join(dbPath, "database.duckdb")
	db, err := sql.Open("duckdb", dbFile)
	if err != nil {
		return nil, fmt.Errorf("open duckdb: %w", err)
	}
	db.SetMaxOpenConns(1)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping duckdb: %w", err)
	}

	c := &Client{db: db}
	if err := c.initSchema(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}
	return c, nil
}

func (c *Client) initSchema(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, err := c.db.ExecContext(ctx, schemaSQL); err != nil {
		return fmt.Errorf("exec schema: %w", err)
	}
	return nil
}

func (c *Client) Close() error {
	return c.db.Close()
}
