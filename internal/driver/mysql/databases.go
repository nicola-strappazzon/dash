package mysql

import (
	"context"
	"fmt"
)

type Database struct {
	Name      string
	Charset   string
	Collation string
	SizeBytes int64
	Tables    int64
}

// Databases lists schemas together with their table count and allocated data
// and index space.
func (m *MySQL) Databases(ctx context.Context) ([]Database, error) {
	const query = `
SELECT
    s.schema_name,
    s.default_character_set_name,
    s.default_collation_name,
    COALESCE(SUM(t.data_length + t.index_length), 0) AS size_bytes,
    COUNT(t.table_name) AS tables
FROM information_schema.schemata s
LEFT JOIN information_schema.tables t ON t.table_schema = s.schema_name
GROUP BY s.schema_name, s.default_character_set_name, s.default_collation_name
ORDER BY size_bytes DESC, s.schema_name`

	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("reading MySQL databases: %w", err)
	}
	defer rows.Close()

	databases := make([]Database, 0)
	for rows.Next() {
		var database Database
		if err := rows.Scan(&database.Name, &database.Charset, &database.Collation, &database.SizeBytes, &database.Tables); err != nil {
			return nil, fmt.Errorf("scanning MySQL database: %w", err)
		}
		databases = append(databases, database)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading MySQL databases: %w", err)
	}
	return databases, nil
}
