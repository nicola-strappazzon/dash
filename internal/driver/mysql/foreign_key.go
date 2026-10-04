package mysql

import (
	"context"
	"fmt"
)

type ForeignKey struct {
	Constraint  string
	Table       string
	Column      string
	OtherTable  string
	OtherColumn string
	OnUpdate    string
	OnDelete    string
}

type ForeignKeyReport struct {
	Outgoing []ForeignKey
	Incoming []ForeignKey
}

// ForeignKeys lists the foreign keys declared by table and the keys from
// other tables that reference it.
func (m *MySQL) ForeignKeys(ctx context.Context, database, table string) (ForeignKeyReport, error) {
	outgoing, err := m.foreignKeys(ctx, `
SELECT
    kcu.constraint_name,
    kcu.table_name,
    kcu.column_name,
    kcu.referenced_table_name,
    kcu.referenced_column_name,
    rc.update_rule,
    rc.delete_rule
FROM information_schema.key_column_usage kcu
INNER JOIN information_schema.referential_constraints rc
    ON rc.constraint_name = kcu.constraint_name
    AND rc.constraint_schema = kcu.table_schema
WHERE kcu.table_schema = ?
  AND kcu.table_name = ?
  AND kcu.referenced_table_name IS NOT NULL
ORDER BY kcu.constraint_name, kcu.ordinal_position`, database, table)
	if err != nil {
		return ForeignKeyReport{}, err
	}

	incoming, err := m.foreignKeys(ctx, `
SELECT
    kcu.constraint_name,
    kcu.table_name,
    kcu.column_name,
    kcu.referenced_table_name,
    kcu.referenced_column_name,
    rc.update_rule,
    rc.delete_rule
FROM information_schema.key_column_usage kcu
INNER JOIN information_schema.referential_constraints rc
    ON rc.constraint_name = kcu.constraint_name
    AND rc.constraint_schema = kcu.table_schema
WHERE kcu.table_schema = ?
  AND kcu.referenced_table_name = ?
ORDER BY kcu.table_name, kcu.constraint_name, kcu.ordinal_position`, database, table)
	if err != nil {
		return ForeignKeyReport{}, err
	}

	return ForeignKeyReport{Outgoing: outgoing, Incoming: incoming}, nil
}

// AllForeignKeys lists every foreign key declared in database.
func (m *MySQL) AllForeignKeys(ctx context.Context, database string) ([]ForeignKey, error) {
	return m.foreignKeys(ctx, `
SELECT
    kcu.constraint_name,
    kcu.table_name,
    kcu.column_name,
    kcu.referenced_table_name,
    kcu.referenced_column_name,
    rc.update_rule,
    rc.delete_rule
FROM information_schema.key_column_usage kcu
INNER JOIN information_schema.referential_constraints rc
    ON rc.constraint_name = kcu.constraint_name
    AND rc.constraint_schema = kcu.table_schema
WHERE kcu.table_schema = ?
  AND kcu.referenced_table_name IS NOT NULL
ORDER BY kcu.table_name, kcu.constraint_name, kcu.ordinal_position`, database)
}

func (m *MySQL) foreignKeys(ctx context.Context, query string, args ...any) ([]ForeignKey, error) {
	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("reading MySQL foreign keys: %w", err)
	}
	defer rows.Close()

	keys := make([]ForeignKey, 0)
	for rows.Next() {
		var key ForeignKey
		if err := rows.Scan(&key.Constraint, &key.Table, &key.Column, &key.OtherTable, &key.OtherColumn, &key.OnUpdate, &key.OnDelete); err != nil {
			return nil, fmt.Errorf("scanning MySQL foreign key: %w", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading MySQL foreign keys: %w", err)
	}
	return keys, nil
}
