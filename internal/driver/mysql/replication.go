package mysql

import (
	"context"
	"fmt"
	"strconv"
)

// Replication returns health checks for a replica. The boolean is false when
// the server is not configured as a replica.
func (m *MySQL) Replication(ctx context.Context) ([]HealthCheck, bool, error) {
	status, isReplica, err := m.replicaStatus(ctx)
	if err != nil {
		return nil, false, err
	}
	if !isReplica {
		return nil, false, nil
	}
	return evaluateReplication(status), true, nil
}

func (m *MySQL) replicaStatus(ctx context.Context) (map[string]string, bool, error) {
	rows, err := m.db.QueryContext(ctx, "SHOW REPLICA STATUS")
	if err != nil {
		return nil, false, fmt.Errorf("reading MySQL replica status: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, false, rows.Err()
	}

	columns, err := rows.Columns()
	if err != nil {
		return nil, false, err
	}
	values := make([]any, len(columns))
	pointers := make([]any, len(columns))
	for i := range values {
		pointers[i] = &values[i]
	}
	if err := rows.Scan(pointers...); err != nil {
		return nil, false, err
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}

	status := make(map[string]string, len(columns))
	for i, column := range columns {
		if values[i] == nil {
			continue
		}
		switch value := values[i].(type) {
		case []byte:
			status[column] = string(value)
		default:
			status[column] = fmt.Sprint(value)
		}
	}
	return status, true, nil
}

func evaluateReplication(status map[string]string) []HealthCheck {
	io := firstStatus(status, "Replica_IO_Running", "Slave_IO_Running")
	sql := firstStatus(status, "Replica_SQL_Running", "Slave_SQL_Running")
	threadStatus := HealthOK
	if io != "Yes" || sql != "Yes" {
		threadStatus = HealthCritical
		if io == "Connecting" && sql == "Yes" {
			threadStatus = HealthWarning
		}
	}
	checks := []HealthCheck{{
		Name:    "Replica IO / SQL",
		Display: fmt.Sprintf("%s / %s", io, sql),
		Status:  threadStatus,
	}}

	rawLag := firstStatus(status, "Seconds_Behind_Source", "Seconds_Behind_Master")
	if rawLag == "" {
		return append(checks, HealthCheck{
			Name:    "Replication lag",
			Display: "NULL",
			Status:  HealthCritical,
		})
	}

	lag, err := strconv.ParseFloat(rawLag, 64)
	if err != nil {
		return append(checks, HealthCheck{
			Name:    "Replication lag",
			Display: rawLag,
			Status:  HealthCritical,
		})
	}
	lagStatus := HealthOK
	if lag > 300 {
		lagStatus = HealthCritical
	} else if lag > 30 {
		lagStatus = HealthWarning
	}
	return append(checks, HealthCheck{
		Name:    "Replication lag",
		Value:   lag,
		Display: fmt.Sprintf("%.0fs", lag),
		Status:  lagStatus,
	})
}

func firstStatus(status map[string]string, names ...string) string {
	for _, name := range names {
		if value, ok := status[name]; ok {
			return value
		}
	}
	return ""
}
