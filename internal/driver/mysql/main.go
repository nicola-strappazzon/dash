// Package mysql provides a MySQL client for dashboard sections.
package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
)

type MySQL struct {
	db *sql.DB
}

const (
	mysqlConnectTimeout = 5 * time.Second
	mysqlReadTimeout    = 30 * time.Second
	mysqlWriteTimeout   = 30 * time.Second
)

// New opens and verifies a MySQL connection.
func New(cfg Config) (*MySQL, error) {
	if cfg.Host == "" {
		return nil, fmt.Errorf("missing MySQL host")
	}

	dsn := mysql.NewConfig()
	dsn.Net = "tcp"
	dsn.Addr = cfg.Host
	dsn.User = cfg.Username
	dsn.Passwd = cfg.Password
	dsn.DBName = cfg.Database
	dsn.Timeout = mysqlConnectTimeout
	dsn.ReadTimeout = mysqlReadTimeout
	dsn.WriteTimeout = mysqlWriteTimeout

	if cfg.InsecureSkipVerify {
		dsn.TLSConfig = "skip-verify"
	} else if cfg.TLS {
		dsn.TLSConfig = "true"
	}

	db, err := sql.Open("mysql", dsn.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("opening MySQL connection: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), mysqlConnectTimeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connecting to MySQL at %s: %w", cfg.Host, err)
	}

	return &MySQL{db: db}, nil
}

func (m *MySQL) DB() *sql.DB {
	return m.db
}

func (m *MySQL) Close() error {
	return m.db.Close()
}
