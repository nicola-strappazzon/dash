package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

type Variable struct {
	Name  string
	Value string
}

// Variables returns global MySQL variables, optionally limited by a MySQL
// LIKE pattern such as "innodb%" or "max%".
func (m *MySQL) Variables(ctx context.Context, like string) ([]Variable, error) {
	query := "SHOW GLOBAL VARIABLES"
	if like != "" {
		query += " LIKE '" + strings.ReplaceAll(like, "'", "''") + "'"
	}

	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("reading MySQL global variables: %w", err)
	}
	defer rows.Close()

	variables := make([]Variable, 0)
	for rows.Next() {
		var variable Variable
		var value sql.NullString
		if err := rows.Scan(&variable.Name, &value); err != nil {
			return nil, fmt.Errorf("scanning MySQL global variable: %w", err)
		}
		variable.Value = value.String
		variables = append(variables, variable)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading MySQL global variables: %w", err)
	}

	sort.Slice(variables, func(i, j int) bool { return variables[i].Name < variables[j].Name })
	return variables, nil
}
