package mysql

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
)

func queryGlobalValues(ctx context.Context, db *sql.DB, statement string, names []string) (map[string]float64, error) {
	placeholders := strings.TrimRight(strings.Repeat("?,", len(names)), ",")
	args := make([]any, len(names))
	for i, name := range names {
		args[i] = name
	}

	rows, err := db.QueryContext(ctx, statement+" WHERE Variable_name IN ("+placeholders+")", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]float64, len(names))
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}
		parsed, err := strconv.ParseFloat(value, 64)
		if err == nil {
			result[name] = parsed
		}
	}
	return result, rows.Err()
}

func values(values map[string]float64, names ...string) (float64, float64, bool) {
	first, firstOK := values[names[0]]
	second, secondOK := values[names[1]]
	return first, second, firstOK && secondOK
}

func statusAndVariable(snapshot HealthSnapshot, statusName, variableName string) (float64, float64, bool) {
	status, statusOK := snapshot.Status[statusName]
	variable, variableOK := snapshot.Variables[variableName]
	return status, variable, statusOK && variableOK
}

func variableValues(snapshot HealthSnapshot, firstName, secondName string) (float64, float64, bool) {
	first, firstOK := snapshot.Variables[firstName]
	second, secondOK := snapshot.Variables[secondName]
	return first, second, firstOK && secondOK
}
