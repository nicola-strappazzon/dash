package mysql

import (
	"context"
	"fmt"
)

// Health collects the server counters once and evaluates the dashboard checks.
func (m *MySQL) Health(ctx context.Context) (HealthReport, error) {
	version, err := m.Version(ctx)
	if err != nil {
		return HealthReport{}, err
	}

	status, err := queryGlobalValues(ctx, m.db, "SHOW GLOBAL STATUS", healthStatusVariables)
	if err != nil {
		return HealthReport{}, fmt.Errorf("reading MySQL global status: %w", err)
	}

	variables, err := queryGlobalValues(ctx, m.db, "SHOW GLOBAL VARIABLES", healthConfigVariables)
	if err != nil {
		return HealthReport{}, fmt.Errorf("reading MySQL global variables: %w", err)
	}

	checks := EvaluateHealth(HealthSnapshot{Status: status, Variables: variables})
	replication, isReplica, err := m.Replication(ctx)
	if err != nil {
		return HealthReport{}, err
	}
	if isReplica {
		checks = append(checks, replication...)
	}

	return HealthReport{
		Checks:  checks,
		Version: version,
	}, nil
}

// Version returns the server version reported by MySQL.
func (m *MySQL) Version(ctx context.Context) (string, error) {
	var version string
	if err := m.db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&version); err != nil {
		return "", fmt.Errorf("reading MySQL version: %w", err)
	}
	return version, nil
}
