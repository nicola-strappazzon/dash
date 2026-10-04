package mysql

// HealthSnapshot contains the raw server counters used to calculate health
// checks. Keeping it separate from EvaluateHealth makes the checks easy to
// test without a live MySQL server.
type HealthSnapshot struct {
	Status    map[string]float64
	Variables map[string]float64
}

var healthStatusVariables = []string{
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
