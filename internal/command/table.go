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
	columns  map[int]string
	title    string
	counters map[int][]string
	identity []int
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

func (t *Table) Title(title string) table.Table {
	t.title = title
	t.Table.Title(title)
	return t
}

// Counter marks a cumulative integer column. Labels restrict it to named rows
// in a key/value table; with no labels, every row in the column is a counter.
func (t *Table) Counter(column int, labels ...string) {
	if t.counters == nil {
		t.counters = make(map[int][]string)
	}
	t.counters[column] = append(t.counters[column], labels...)
}

// Identity selects the columns that identify a series across samples.
// Tables use their first column by default.
func (t *Table) Identity(columns ...int) { t.identity = columns }
