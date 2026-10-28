package mysql

import "testing"

func TestStatusVariableIsCounter(t *testing.T) {
	for _, name := range []string{"Table_locks_immediate", "Table_locks_waited", "Threads_created", "Com_select", "Handler_read_next", "Connection_errors_internal", "Innodb_rows_read", "Innodb_row_lock_time", "Bytes_sent"} {
		if !(StatusVariable{Name: name}).IsCounter() {
			t.Errorf("%s should be a counter", name)
		}
	}
	for _, name := range []string{"Threads_connected", "Threads_running", "Threads_cached", "Open_tables", "Innodb_row_lock_current_waits", "Innodb_row_lock_time_avg", "Innodb_row_lock_time_max", "Innodb_buffer_pool_pages_dirty", "Uptime", "Unknown_total"} {
		if (StatusVariable{Name: name}).IsCounter() {
			t.Errorf("%s should remain unchanged", name)
		}
	}
}
