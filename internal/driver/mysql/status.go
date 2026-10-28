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

// IsCounter classifies documented cumulative status variables. Unknown names
// are left untouched; a numeric value alone does not imply a counter.
// https://dev.mysql.com/doc/refman/8.4/en/server-status-variables.html
func (v StatusVariable) IsCounter() bool {
	name := strings.ToLower(v.Name)
	if strings.HasPrefix(name, "com_") || strings.HasPrefix(name, "handler_") ||
		strings.HasPrefix(name, "connection_errors_") {
		return true
	}
	switch name {
	case "aborted_clients", "aborted_connects", "bytes_received", "bytes_sent",
		"connections", "created_tmp_disk_tables", "created_tmp_files", "created_tmp_tables",
		"key_read_requests", "key_reads", "key_write_requests", "key_writes",
		"opened_files", "opened_tables", "opened_table_definitions", "queries", "questions",
		"select_full_join", "select_full_range_join", "select_range", "select_range_check", "select_scan",
		"slow_launch_threads", "slow_queries", "sort_merge_passes", "sort_range", "sort_rows", "sort_scan",
		"table_locks_immediate", "table_locks_waited", "table_open_cache_hits", "table_open_cache_misses",
		"table_open_cache_overflows", "threads_created",
		"innodb_buffer_pool_read_ahead", "innodb_buffer_pool_read_ahead_evicted", "innodb_buffer_pool_read_ahead_rnd",
		"innodb_buffer_pool_read_requests", "innodb_buffer_pool_reads", "innodb_buffer_pool_wait_free", "innodb_buffer_pool_write_requests",
		"innodb_data_fsyncs", "innodb_data_read", "innodb_data_reads", "innodb_data_writes", "innodb_data_written",
		"innodb_dblwr_pages_written", "innodb_dblwr_writes", "innodb_log_waits", "innodb_log_write_requests", "innodb_log_writes",
		"innodb_os_log_fsyncs", "innodb_os_log_written", "innodb_pages_created", "innodb_pages_read", "innodb_pages_written",
		"innodb_row_lock_time", "innodb_row_lock_waits", "innodb_rows_deleted", "innodb_rows_inserted", "innodb_rows_read", "innodb_rows_updated",
		"innodb_system_rows_deleted", "innodb_system_rows_inserted", "innodb_system_rows_read", "innodb_system_rows_updated":
		return true
	}
	return false
}
