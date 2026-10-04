package mysql

import (
	"context"
	"database/sql"
	"fmt"
)

type Process struct {
	ID       int64
	User     string
	Host     string
	Database string
	Command  string
	Time     int64
	State    string
	Info     string
}

// Processlist lists server processes with active statements, optionally
// including idle connections.
func (m *MySQL) Processlist(ctx context.Context, includeIdle bool, minTime int64) ([]Process, error) {
	rows, err := m.db.QueryContext(ctx, `
SELECT ID, USER, HOST, DB, COMMAND, TIME, STATE, INFO
FROM information_schema.PROCESSLIST
WHERE ID <> CONNECTION_ID()
ORDER BY TIME DESC, ID`)
	if err != nil {
		return nil, fmt.Errorf("reading MySQL processlist: %w", err)
	}
	defer rows.Close()

	processes := make([]Process, 0)
	for rows.Next() {
		var (
			process  Process
			database sql.NullString
			state    sql.NullString
			info     sql.NullString
		)
		if err := rows.Scan(&process.ID, &process.User, &process.Host, &database, &process.Command, &process.Time, &state, &info); err != nil {
			return nil, fmt.Errorf("scanning MySQL process: %w", err)
		}
		process.Database = database.String
		process.State = state.String
		process.Info = info.String

		if process.Command == "Sleep" && !includeIdle {
			continue
		}
		if process.Time < minTime {
			continue
		}
		if process.Info == "" {
			continue
		}
		processes = append(processes, process)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading MySQL processlist: %w", err)
	}
	return processes, nil
}
