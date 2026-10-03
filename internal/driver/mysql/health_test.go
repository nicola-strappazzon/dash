package mysql

import (
	"testing"
	"time"
)

func TestEvaluateHealth(t *testing.T) {
	checks := EvaluateHealth(HealthSnapshot{
		Status: map[string]float64{
			"Innodb_buffer_pool_read_requests": 1000,
			"Innodb_buffer_pool_reads":         20,
			"Threads_connected":                850,
			"Created_tmp_disk_tables":          30,
			"Created_tmp_tables":               100,
			"Threads_created":                  600,
			"Connections":                      1000,
			"Innodb_buffer_pool_pages_dirty":   800,
			"Innodb_buffer_pool_pages_total":   1000,
			"Innodb_log_waits":                 25,
			"Innodb_log_writes":                100,
		},
		Variables: map[string]float64{"max_connections": 1000},
	})

	want := []Status{
		HealthWarning,
		HealthCritical,
		HealthWarning,
		HealthCritical,
		HealthWarning,
		HealthCritical,
	}
	if len(checks) != len(want) {
		t.Fatalf("got %d checks, want %d", len(checks), len(want))
	}
	for i, status := range want {
		if checks[i].Status != status {
			t.Errorf("check %q status = %q, want %q", checks[i].Name, checks[i].Status, status)
		}
	}
}

func TestEvaluateHealthSkipsUnavailableMetrics(t *testing.T) {
	checks := EvaluateHealth(HealthSnapshot{})
	if len(checks) != 0 {
		t.Fatalf("got %d checks, want none", len(checks))
	}
}

func TestEvaluateAdvancedHealth(t *testing.T) {
	checks := EvaluateHealth(HealthSnapshot{
		Status: map[string]float64{
			"Threads_cached":        5,
			"Threads_created":       100,
			"Open_files":            90,
			"Sort_merge_passes":     25,
			"Sort_scan":             50,
			"Sort_range":            50,
			"Uptime":                600,
			"Innodb_os_log_written": 100,
		},
		Variables: map[string]float64{
			"open_files_limit":         100,
			"tmp_table_size":           100 * 1024 * 1024,
			"max_heap_table_size":      50 * 1024 * 1024,
			"temptable_max_ram":        1024 * 1024 * 1024,
			"innodb_redo_log_capacity": 1024 * 1024 * 1024,
		},
	})

	want := []Status{HealthWarning, HealthWarning, HealthCritical, HealthWarning, HealthWarning}
	if len(checks) != len(want) {
		t.Fatalf("got %d checks, want %d", len(checks), len(want))
	}
	for i, status := range want {
		if checks[i].Status != status {
			t.Errorf("check %q status = %q, want %q", checks[i].Name, checks[i].Status, status)
		}
	}
}

func TestEvaluateHistoryListLength(t *testing.T) {
	if got := evaluateHistoryListLength(10_001).Status; got != HealthWarning {
		t.Errorf("status = %q, want %q", got, HealthWarning)
	}
	if got := evaluateHistoryListLength(100_001).Status; got != HealthCritical {
		t.Errorf("status = %q, want %q", got, HealthCritical)
	}
}

func TestDeadlockTracker(t *testing.T) {
	tracker := &DeadlockTracker{}
	start := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)

	first := tracker.Check(10, start)
	if first.Display != "10 since startup" || first.Status != HealthOK {
		t.Fatalf("first check = %#v", first)
	}

	second := tracker.Check(11, start.Add(5*time.Minute))
	if second.Status != HealthWarning || second.Display != "+1 (0.20/min)" {
		t.Fatalf("second check = %#v", second)
	}

	reset := tracker.Check(2, start.Add(10*time.Minute))
	if reset.Status != HealthOK || reset.Display != "2 since restart" {
		t.Fatalf("reset check = %#v", reset)
	}
}
