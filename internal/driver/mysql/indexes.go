package mysql

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

type IndexColumn struct {
	Position   int64
	Name       string
	Expression string
	Collation  string
	SubPart    int64
	Nullable   bool
}

type Index struct {
	Name        string
	Type        string
	Unique      bool
	Visible     bool
	Cardinality int64
	SizeBytes   int64
	SizeKnown   bool
	Columns     []IndexColumn
}

// Indexes lists each index and its indexed columns for a table. Index size is
// read when mysql.innodb_index_stats is accessible.
func (m *MySQL) Indexes(ctx context.Context, database, table string) ([]Index, error) {
	const query = `
SELECT
    index_name, index_type, non_unique, is_visible,
    COALESCE(cardinality, 0),
    seq_in_index, COALESCE(column_name, ''), COALESCE(expression, ''),
    COALESCE(collation, ''), COALESCE(sub_part, 0), COALESCE(nullable, '')
FROM information_schema.statistics
WHERE table_schema = ? AND table_name = ?
ORDER BY index_name, seq_in_index`

	rows, err := m.db.QueryContext(ctx, query, database, table)
	if err != nil {
		return nil, fmt.Errorf("reading MySQL indexes: %w", err)
	}
	defer rows.Close()

	byName := make(map[string]*Index)
	for rows.Next() {
		var (
			name, indexType, visible, columnName, expression, collation, nullable string
			nonUnique                                                             int64
			cardinality, position, subPart                                        int64
		)
		if err := rows.Scan(&name, &indexType, &nonUnique, &visible, &cardinality,
			&position, &columnName, &expression, &collation, &subPart, &nullable); err != nil {
			return nil, fmt.Errorf("scanning MySQL index: %w", err)
		}

		index := byName[name]
		if index == nil {
			index = &Index{
				Name: name, Type: indexType, Unique: nonUnique == 0, Visible: visible == "YES",
				Cardinality: cardinality,
			}
			byName[name] = index
		}
		index.Columns = append(index.Columns, IndexColumn{
			Position: position, Name: columnName, Expression: expression, Collation: collation,
			SubPart: subPart, Nullable: nullable == "YES",
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading MySQL indexes: %w", err)
	}

	// This system table is InnoDB-specific and may require additional grants.
	// Index metadata remains useful without it; individual unknown sizes are
	// represented by SizeKnown=false rather than a misleading zero.
	const sizeQuery = `
SELECT index_name, stat_value * @@innodb_page_size
FROM mysql.innodb_index_stats
WHERE database_name = ? AND table_name = ? AND stat_name = 'size'`
	if sizeRows, err := m.db.QueryContext(ctx, sizeQuery, database, table); err == nil {
		defer sizeRows.Close()
		for sizeRows.Next() {
			var name string
			var size int64
			if err := sizeRows.Scan(&name, &size); err == nil {
				if index := byName[name]; index != nil {
					index.SizeBytes = size
					index.SizeKnown = true
				}
			}
		}
	}

	indexes := make([]Index, 0, len(byName))
	for _, index := range byName {
		indexes = append(indexes, *index)
	}
	sort.Slice(indexes, func(i, j int) bool {
		if indexes[i].Name == "PRIMARY" {
			return true
		}
		if indexes[j].Name == "PRIMARY" {
			return false
		}
		return indexes[i].Name < indexes[j].Name
	})
	return indexes, nil
}

func FormatIndexColumns(columns []IndexColumn) string {
	values := make([]string, 0, len(columns))
	for _, column := range columns {
		name := column.Name
		if name == "" {
			name = "(" + column.Expression + ")"
		}
		if column.SubPart > 0 {
			name += fmt.Sprintf("(%d)", column.SubPart)
		}
		switch column.Collation {
		case "A":
			name += " ASC"
		case "D":
			name += " DESC"
		}
		if column.Nullable {
			name += " NULL"
		} else {
			name += " NOT NULL"
		}
		values = append(values, fmt.Sprintf("%d:%s", column.Position, name))
	}
	return strings.Join(values, ", ")
}
