package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Status string

const (
	HealthOK       Status = "ok"
	HealthWarning  Status = "warning"
	HealthCritical Status = "critical"
)

type HealthCheck struct {
	Name      string
	Value     float64
	Display   string
	Status    Status
	Threshold string
}

type HealthReport struct {
	Checks       []HealthCheck
	Version      string
	Deadlocks    uint64
	HasDeadlocks bool
}

// DeadlockTracker turns MySQL's cumulative Innodb_deadlocks counter into a
// rate between dashboard refreshes.
type DeadlockTracker struct {
	last       uint64
	observedAt time.Time
	ready      bool
}

func (t *DeadlockTracker) Check(total uint64, now time.Time) HealthCheck {
	if !t.ready {
		t.last = total
		t.observedAt = now
		t.ready = true
		return HealthCheck{
			Name:      "InnoDB deadlocks",
			Display:   fmt.Sprintf("%d since startup", total),
			Status:    HealthOK,
			Threshold: "rate available with --watch",
		}
	}

	elapsed := now.Sub(t.observedAt)
	if total < t.last {
		t.last = total
		t.observedAt = now
		return HealthCheck{
			Name:      "InnoDB deadlocks",
			Display:   fmt.Sprintf("%d since restart", total),
			Status:    HealthOK,
			Threshold: "counter reset detected",
		}
	}
	delta := total - t.last
	rate := 0.0
	if elapsed > 0 {
		rate = float64(delta) / elapsed.Minutes()
	}
	t.last = total
	t.observedAt = now

	status := HealthOK
	if rate >= 1 {
		status = HealthCritical
	} else if rate > 0 {
		status = HealthWarning
	}
	return HealthCheck{
		Name:      "InnoDB deadlocks",
		Value:     rate,
		Display:   fmt.Sprintf("+%d (%.2f/min)", delta, rate),
		Status:    status,
		Threshold: "warning >0/min, critical >=1/min",
	}
}

// HealthSnapshot contains the raw server counters used to calculate health
// checks. Keeping it separate from EvaluateHealth makes the thresholds easy
// to test without a live MySQL server.
type HealthSnapshot struct {
	Status    map[string]float64
	Variables map[string]float64
}

var StatusVariables = []string{
	"Innodb_buffer_pool_read_requests",
	"Innodb_buffer_pool_reads",
	"Threads_connected",
	"Threads_running",
	"Max_used_connections",
	"Created_tmp_disk_tables",
	"Created_tmp_tables",
	"Threads_created",
	"Connections",
	"Innodb_buffer_pool_pages_dirty",
	"Innodb_buffer_pool_pages_total",
	"Innodb_log_waits",
	"Innodb_log_writes",
	"Innodb_deadlocks",
	"Threads_cached",
	"Open_files",
	"Sort_merge_passes",
	"Sort_scan",
	"Sort_range",
	"Uptime",
	"Innodb_os_log_written",
}

var healthConfigVariables = []string{
	"max_connections",
	"open_files_limit",
	"tmp_table_size",
	"max_heap_table_size",
	"temptable_max_ram",
	"innodb_redo_log_capacity",
}

var historyListLengthPattern = regexp.MustCompile(`History list length (\d+)`)

// Health collects the server counters once and evaluates the dashboard checks.
func (m *MySQL) Health(ctx context.Context) (HealthReport, error) {
	version, err := m.Version(ctx)
	if err != nil {
		return HealthReport{}, err
	}

	status, err := queryGlobalValues(ctx, m.db, "SHOW GLOBAL STATUS", StatusVariables)
	if err != nil {
		return HealthReport{}, fmt.Errorf("reading MySQL global status: %w", err)
	}

	variables, err := queryGlobalValues(ctx, m.db, "SHOW GLOBAL VARIABLES", healthConfigVariables)
	if err != nil {
		return HealthReport{}, fmt.Errorf("reading MySQL global variables: %w", err)
	}

	checks := EvaluateHealth(HealthSnapshot{Status: status, Variables: variables})
	historyListLength, found, err := m.historyListLength(ctx)
	if err != nil {
		return HealthReport{}, err
	}
	if found {
		checks = append(checks, evaluateHistoryListLength(historyListLength))
	}
	replication, isReplica, err := m.Replication(ctx)
	if err != nil {
		return HealthReport{}, err
	}
	if isReplica {
		checks = append(checks, replication...)
	}

	deadlocks, hasDeadlocks := status["Innodb_deadlocks"]
	return HealthReport{
		Checks:       checks,
		Version:      version,
		Deadlocks:    uint64(deadlocks),
		HasDeadlocks: hasDeadlocks,
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

func (m *MySQL) historyListLength(ctx context.Context) (int64, bool, error) {
	var engineType, name, output string
	if err := m.db.QueryRowContext(ctx, "SHOW ENGINE INNODB STATUS").Scan(&engineType, &name, &output); err != nil {
		return 0, false, fmt.Errorf("reading InnoDB status: %w", err)
	}

	match := historyListLengthPattern.FindStringSubmatch(output)
	if match == nil {
		return 0, false, nil
	}
	value, err := strconv.ParseInt(match[1], 10, 64)
	if err != nil {
		return 0, false, fmt.Errorf("parsing InnoDB history list length: %w", err)
	}
	return value, true, nil
}

// EvaluateHealth calculates the checks that can be derived from a snapshot.
// Checks whose source values are unavailable are omitted.
func EvaluateHealth(snapshot HealthSnapshot) []HealthCheck {
	var checks []HealthCheck

	if requests, reads, ok := values(snapshot.Status, "Innodb_buffer_pool_read_requests", "Innodb_buffer_pool_reads"); ok && requests > 0 {
		hitRate := (requests - reads) * 100 / requests
		checks = append(checks, percentageCheck("Buffer pool hit rate", hitRate, 99, 90, true))
	}

	if connected, maxConnections, ok := statusAndVariable(snapshot, "Threads_connected", "max_connections"); ok && maxConnections > 0 {
		usage := connected * 100 / maxConnections
		status := HealthOK
		if usage > 80 {
			status = HealthCritical
		} else if usage > 70 {
			status = HealthWarning
		}
		checks = append(checks, HealthCheck{
			Name:      "Connections",
			Value:     usage,
			Display:   fmt.Sprintf("%.0f / %.0f (%.1f%%)", connected, maxConnections, usage),
			Status:    status,
			Threshold: "warning >70%, critical >80%",
		})
	}

	if running, ok := snapshot.Status["Threads_running"]; ok {
		status := HealthOK
		if running > 50 {
			status = HealthCritical
		} else if running > 10 {
			status = HealthWarning
		}
		checks = append(checks, HealthCheck{
			Name:      "Running threads",
			Value:     running,
			Display:   fmt.Sprintf("%.0f", running),
			Status:    status,
			Threshold: "warning >10, critical >50",
		})
	}

	if peak, maxConnections, ok := statusAndVariable(snapshot, "Max_used_connections", "max_connections"); ok && maxConnections > 0 {
		usage := peak * 100 / maxConnections
		status := HealthOK
		if usage > 80 {
			status = HealthCritical
		} else if usage > 70 {
			status = HealthWarning
		}
		checks = append(checks, HealthCheck{
			Name:      "Peak connections",
			Value:     usage,
			Display:   fmt.Sprintf("%.0f / %.0f (%.1f%%)", peak, maxConnections, usage),
			Status:    status,
			Threshold: "warning >70%, critical >80%",
		})
	}

	if disk, total, ok := values(snapshot.Status, "Created_tmp_disk_tables", "Created_tmp_tables"); ok && total > 0 {
		percentage := disk * 100 / total
		status := HealthOK
		if percentage > 25 {
			status = HealthWarning
		}
		checks = append(checks, HealthCheck{
			Name:      "Temp tables on disk",
			Value:     percentage,
			Display:   fmt.Sprintf("%.2f%%", percentage),
			Status:    status,
			Threshold: "warning >25%",
		})
	}

	if created, connections, ok := values(snapshot.Status, "Threads_created", "Connections"); ok && connections > 0 {
		hitRate := 100 - created*100/connections
		checks = append(checks, percentageCheck("Thread cache hit rate", hitRate, 90, 50, true))
	}

	if cached, created, ok := values(snapshot.Status, "Threads_cached", "Threads_created"); ok && created > 0 {
		ratio := cached * 100 / created
		status := HealthOK
		if ratio < 10 {
			status = HealthWarning
		}
		checks = append(checks, HealthCheck{
			Name:      "Thread cache ratio",
			Value:     ratio,
			Display:   fmt.Sprintf("%.2f%%", ratio),
			Status:    status,
			Threshold: "warning <10%",
		})
	}

	if dirty, total, ok := values(snapshot.Status, "Innodb_buffer_pool_pages_dirty", "Innodb_buffer_pool_pages_total"); ok && total > 0 {
		ratio := dirty * 100 / total
		status := HealthOK
		if ratio >= 75 {
			status = HealthWarning
		}
		checks = append(checks, HealthCheck{
			Name:      "InnoDB dirty pages",
			Value:     ratio,
			Display:   fmt.Sprintf("%.2f%%", ratio),
			Status:    status,
			Threshold: "warning >=75%",
		})
	}

	if waits, writes, ok := values(snapshot.Status, "Innodb_log_waits", "Innodb_log_writes"); ok && writes > 0 {
		ratio := waits * 100 / writes
		status := HealthOK
		if ratio > 20 {
			status = HealthCritical
		} else if ratio >= 5 {
			status = HealthWarning
		}
		checks = append(checks, HealthCheck{
			Name:      "InnoDB log waits",
			Value:     ratio,
			Display:   fmt.Sprintf("%.2f%%", ratio),
			Status:    status,
			Threshold: "warning 5-20%, critical >20%",
		})
	}

	if openFiles, limit, ok := statusAndVariable(snapshot, "Open_files", "open_files_limit"); ok && limit > 0 {
		ratio := openFiles * 100 / limit
		status := HealthOK
		if ratio >= 85 {
			status = HealthWarning
		}
		checks = append(checks, HealthCheck{
			Name:      "Open files",
			Value:     ratio,
			Display:   fmt.Sprintf("%.0f / %.0f (%.2f%%)", openFiles, limit, ratio),
			Status:    status,
			Threshold: "warning >=85%",
		})
	}

	if passes, scans, ok := values(snapshot.Status, "Sort_merge_passes", "Sort_scan"); ok {
		if ranges, rangeOK := snapshot.Status["Sort_range"]; rangeOK && scans+ranges > 0 {
			ratio := passes * 100 / (scans + ranges)
			status := HealthOK
			if ratio >= 25 {
				status = HealthCritical
			} else if ratio >= 10 {
				status = HealthWarning
			}
			checks = append(checks, HealthCheck{
				Name:      "Sort merge passes",
				Value:     ratio,
				Display:   fmt.Sprintf("%.2f%%", ratio),
				Status:    status,
				Threshold: "warning >=10%, critical >=25%",
			})
		}
	}

	if tmpTableSize, maxHeapSize, ok := variableValues(snapshot, "tmp_table_size", "max_heap_table_size"); ok {
		tempTableRAM := snapshot.Variables["temptable_max_ram"]
		status := HealthOK
		if maxHeapSize < tmpTableSize || (tempTableRAM > 0 && tempTableRAM < 2*1024*1024*1024) {
			status = HealthWarning
		}
		checks = append(checks, HealthCheck{
			Name:      "Temp table config",
			Display:   fmt.Sprintf("tmp %.0fMB, heap %.0fMB, RAM %.0fMB", tmpTableSize/1024/1024, maxHeapSize/1024/1024, tempTableRAM/1024/1024),
			Status:    status,
			Threshold: "heap >= tmp, RAM >=2GB",
		})
	}

	if uptime, written, ok := values(snapshot.Status, "Uptime", "Innodb_os_log_written"); ok && uptime > 0 && written > 0 {
		if capacity, capacityOK := snapshot.Variables["innodb_redo_log_capacity"]; capacityOK && capacity > 0 {
			minutes := (uptime / 60) * capacity / written
			status := HealthOK
			if minutes < 45 || minutes > 75 {
				status = HealthWarning
			}
			checks = append(checks, HealthCheck{
				Name:      "InnoDB redo log",
				Value:     minutes,
				Display:   fmt.Sprintf("%.1f min", minutes),
				Status:    status,
				Threshold: "warning <45 or >75 min",
			})
		}
	}

	return checks
}

func evaluateHistoryListLength(length int64) HealthCheck {
	status := HealthOK
	if length > 100_000 {
		status = HealthCritical
	} else if length > 10_000 {
		status = HealthWarning
	}
	return HealthCheck{
		Name:      "History list length",
		Value:     float64(length),
		Display:   fmt.Sprintf("%d", length),
		Status:    status,
		Threshold: "warning >10k, critical >100k",
	}
}

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

func percentageCheck(name string, value, warning, critical float64, higherIsBetter bool) HealthCheck {
	status := HealthOK
	if higherIsBetter {
		if value < critical {
			status = HealthCritical
		} else if value < warning {
			status = HealthWarning
		}
	}
	return HealthCheck{
		Name:      name,
		Value:     value,
		Display:   fmt.Sprintf("%.2f%%", value),
		Status:    status,
		Threshold: fmt.Sprintf("warning <%g%%, critical <%g%%", warning, critical),
	}
}
