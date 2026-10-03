package mysql

import (
	"context"
	"fmt"
)

type Table struct {
	Name                 string
	Engine               string
	Rows                 int64
	DataBytes            int64
	IndexBytes           int64
	FreeBytes            int64
	FragmentationPercent float64
}

// Tables lists base tables in database, ordered by allocated data and index
// space.
func (m *MySQL) Tables(ctx context.Context, database string) ([]Table, error) {
	const query = `
SELECT
    t.table_name,
    COALESCE(t.engine, '') AS engine,
    COALESCE(t.table_rows, 0) AS row_count,
    COALESCE(t.data_length, 0) AS data_bytes,
    COALESCE(t.index_length, 0) AS index_bytes,
    COALESCE(t.data_free, 0) AS free_bytes,
    COALESCE(ROUND(
        COALESCE(t.data_free, 0) /
        NULLIF(t.data_length + t.index_length + t.data_free, 0) * 100,
        1
    ), 0) AS fragmentation_percent
FROM information_schema.tables t
WHERE t.table_schema = ?
  AND t.table_type = 'BASE TABLE'
ORDER BY data_bytes + index_bytes DESC, t.table_name`

	rows, err := m.db.QueryContext(ctx, query, database)
	if err != nil {
		return nil, fmt.Errorf("reading MySQL tables: %w", err)
	}
	defer rows.Close()

	tables := make([]Table, 0)
	for rows.Next() {
		var table Table
		if err := rows.Scan(
			&table.Name,
			&table.Engine,
			&table.Rows,
			&table.DataBytes,
			&table.IndexBytes,
			&table.FreeBytes,
			&table.FragmentationPercent,
		); err != nil {
			return nil, fmt.Errorf("scanning MySQL table: %w", err)
		}
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading MySQL tables: %w", err)
	}
	return tables, nil
}
