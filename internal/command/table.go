package command

import (
	"strconv"
	"strings"

	"github.com/nicola-strappazzon/go-table"
)

// Table retains column names for structured output, including empty results.
// Rendering, sorting and filtering remain the responsibility of go-table.
type Table struct {
	table.Table
	columns map[int]string
}

func NewTable() *Table {
	return &Table{Table: table.New(), columns: make(map[int]string)}
}

func (t *Table) Column(index int, column table.Column) table.Table {
	t.columns[index] = strings.TrimSuffix(column.Name, table.DOWN_ARROW)
	t.Table.Column(index, column)
	return t
}

func (t *Table) outputColumns() []string {
	columns := make([]string, len(t.columns))
	for i := range columns {
		columns[i] = t.columns[i]
		if columns[i] == "" {
			columns[i] = strconv.Itoa(i)
		}
	}
	return columns
}
