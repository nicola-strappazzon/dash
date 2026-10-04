package mysql

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

type AutoIncrement struct {
	Table        string
	Column       string
	ColumnType   string
	CurrentValue uint64
	MaxValue     uint64
	UsagePercent float64
	Remaining    uint64
}

// AutoIncrements lists AUTO_INCREMENT columns in database, ordered by the
// percentage of their numeric range already consumed.
func (m *MySQL) AutoIncrements(ctx context.Context, database string) ([]AutoIncrement, error) {
	rows, err := m.db.QueryContext(ctx, `
SELECT
    c.table_name,
    c.column_name,
    c.column_type,
    COALESCE(t.auto_increment, 0)
FROM information_schema.columns c
INNER JOIN information_schema.tables t
    ON t.table_schema = c.table_schema
    AND t.table_name = c.table_name
WHERE c.table_schema = ?
  AND c.extra LIKE '%auto_increment%'
  AND t.table_type = 'BASE TABLE'`, database)
	if err != nil {
		return nil, fmt.Errorf("reading MySQL AUTO_INCREMENT columns: %w", err)
	}
	defer rows.Close()

	columns := make([]AutoIncrement, 0)
	for rows.Next() {
		var column AutoIncrement
		if err := rows.Scan(&column.Table, &column.Column, &column.ColumnType, &column.CurrentValue); err != nil {
			return nil, fmt.Errorf("scanning MySQL AUTO_INCREMENT column: %w", err)
		}
		column.calculateCapacity()
		columns = append(columns, column)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading MySQL AUTO_INCREMENT columns: %w", err)
	}

	sort.Slice(columns, func(i, j int) bool {
		return columns[i].UsagePercent > columns[j].UsagePercent
	})
	return columns, nil
}

func (column *AutoIncrement) calculateCapacity() {
	column.MaxValue = autoIncrementMax(column.ColumnType)
	column.UsagePercent = float64(column.CurrentValue) * 100 / float64(column.MaxValue)
	column.Remaining = column.MaxValue - column.CurrentValue
}

func autoIncrementMax(columnType string) uint64 {
	columnType = strings.ToLower(columnType)
	unsigned := strings.Contains(columnType, "unsigned")

	switch {
	case strings.Contains(columnType, "tinyint"):
		if unsigned {
			return 255
		}
		return 127
	case strings.Contains(columnType, "smallint"):
		if unsigned {
			return 65535
		}
		return 32767
	case strings.Contains(columnType, "mediumint"):
		if unsigned {
			return 16777215
		}
		return 8388607
	case strings.Contains(columnType, "bigint"):
		if unsigned {
			return 18446744073709551615
		}
		return 9223372036854775807
	default:
		if unsigned {
			return 4294967295
		}
		return 2147483647
	}
}
