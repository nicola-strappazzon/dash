package clickhouse

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type ClickHouse struct {
	conn driver.Conn
}

type Config struct {
	Host               string
	Username           string
	Password           string
	TLS                bool
	InsecureSkipVerify bool
}

const (
	clickhouseDialTimeout = 5 * time.Second
	clickhouseReadTimeout = 30 * time.Second
)

func New(cfg Config) (*ClickHouse, error) {
	options := &clickhouse.Options{
		Addr:     []string{cfg.Host},
		Protocol: clickhouse.Native,
		Auth: clickhouse.Auth{
			Username: cfg.Username,
			Password: cfg.Password,
		},
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
		DialTimeout: clickhouseDialTimeout,
		ReadTimeout: clickhouseReadTimeout,
	}

	if cfg.TLS || cfg.InsecureSkipVerify {
		options.TLS = &tls.Config{InsecureSkipVerify: cfg.InsecureSkipVerify}
	}

	conn, err := clickhouse.Open(options)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), clickhouseDialTimeout)
	defer cancel()

	if err := conn.Ping(ctx); err != nil {
		return nil, err
	}

	return &ClickHouse{conn: conn}, nil
}

func (ch *ClickHouse) Conn() driver.Conn {
	return ch.conn
}
