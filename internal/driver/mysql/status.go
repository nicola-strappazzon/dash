package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

type StatusVariable struct {
	Name  string
	Value string
}

// GlobalStatus returns global MySQL status counters, optionally limited by a
// MySQL LIKE pattern such as "Innodb%" or "Threads%".
func (m *MySQL) GlobalStatus(ctx context.Context, like string) ([]StatusVariable, error) {
	query := "SHOW GLOBAL STATUS"
	if like != "" {
		query += " LIKE '" + strings.ReplaceAll(like, "'", "''") + "'"
	}

	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("reading MySQL global status: %w", err)
	}
	defer rows.Close()

	status := make([]StatusVariable, 0)
	for rows.Next() {
		var variable StatusVariable
		var value sql.NullString
		if err := rows.Scan(&variable.Name, &value); err != nil {
			return nil, fmt.Errorf("scanning MySQL global status: %w", err)
		}
		variable.Value = value.String
		status = append(status, variable)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading MySQL global status: %w", err)
	}

	sort.Slice(status, func(i, j int) bool { return status[i].Name < status[j].Name })
	return status, nil
}
