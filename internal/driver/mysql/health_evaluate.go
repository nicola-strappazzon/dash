package mysql

import "fmt"

// EvaluateHealth calculates the checks that can be derived from a snapshot.
// Checks whose source values are unavailable are omitted.
func EvaluateHealth(snapshot HealthSnapshot) []HealthCheck {
	var checks []HealthCheck

	if requests, reads, ok := values(snapshot.Status, "Innodb_buffer_pool_read_requests", "Innodb_buffer_pool_reads"); ok && requests > 0 {
		hitRate := (requests - reads) * 100 / requests
		checks = append(checks, percentageCheck("Buffer pool hit rate", hitRate, 99, 90))
	}

	if connected, maxConnections, ok := statusAndVariable(snapshot, "Threads_connected", "max_connections"); ok && maxConnections > 0 {
		usage := connected * 100 / maxConnections
		checks = append(checks, connectionCheck("Connections", connected, maxConnections, usage))
	}

	if running, ok := snapshot.Status["Threads_running"]; ok {
		status := HealthOK
		if running > 50 {
			status = HealthCritical
		} else if running > 10 {
			status = HealthWarning
		}
		checks = append(checks, HealthCheck{Name: "Running threads", Value: running, Display: fmt.Sprintf("%.0f", running), Status: status})
	}

	if peak, maxConnections, ok := statusAndVariable(snapshot, "Max_used_connections", "max_connections"); ok && maxConnections > 0 {
		usage := peak * 100 / maxConnections
		checks = append(checks, connectionCheck("Peak connections", peak, maxConnections, usage))
	}

	if disk, total, ok := values(snapshot.Status, "Created_tmp_disk_tables", "Created_tmp_tables"); ok && total > 0 {
		percentage := disk * 100 / total
		status := HealthOK
		if percentage > 25 {
			status = HealthWarning
		}
		checks = append(checks, HealthCheck{Name: "Temp tables on disk", Value: percentage, Display: fmt.Sprintf("%.2f%%", percentage), Status: status})
	}

	if created, connections, ok := values(snapshot.Status, "Threads_created", "Connections"); ok && connections > 0 {
		hitRate := 100 - created*100/connections
		checks = append(checks, percentageCheck("Thread cache hit rate", hitRate, 90, 50))
	}

	if cached, created, ok := values(snapshot.Status, "Threads_cached", "Threads_created"); ok && created > 0 {
		ratio := cached * 100 / created
		status := HealthOK
		if ratio < 10 {
			status = HealthWarning
		}
		checks = append(checks, HealthCheck{Name: "Thread cache ratio", Value: ratio, Display: fmt.Sprintf("%.2f%%", ratio), Status: status})
	}

	if dirty, total, ok := values(snapshot.Status, "Innodb_buffer_pool_pages_dirty", "Innodb_buffer_pool_pages_total"); ok && total > 0 {
		ratio := dirty * 100 / total
		status := HealthOK
		if ratio >= 75 {
			status = HealthWarning
		}
		checks = append(checks, HealthCheck{Name: "InnoDB dirty pages", Value: ratio, Display: fmt.Sprintf("%.2f%%", ratio), Status: status})
	}

	if waits, writes, ok := values(snapshot.Status, "Innodb_log_waits", "Innodb_log_writes"); ok && writes > 0 {
		ratio := waits * 100 / writes
		status := HealthOK
		if ratio > 20 {
			status = HealthCritical
		} else if ratio >= 5 {
			status = HealthWarning
		}
		checks = append(checks, HealthCheck{Name: "InnoDB log waits", Value: ratio, Display: fmt.Sprintf("%.2f%%", ratio), Status: status})
	}

	if openFiles, limit, ok := statusAndVariable(snapshot, "Open_files", "open_files_limit"); ok && limit > 0 {
		ratio := openFiles * 100 / limit
		status := HealthOK
		if ratio >= 85 {
			status = HealthWarning
		}
		checks = append(checks, HealthCheck{Name: "Open files", Value: ratio, Display: fmt.Sprintf("%.0f / %.0f (%.2f%%)", openFiles, limit, ratio), Status: status})
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
			checks = append(checks, HealthCheck{Name: "Sort merge passes", Value: ratio, Display: fmt.Sprintf("%.2f%%", ratio), Status: status})
		}
	}

	if tmpTableSize, maxHeapSize, ok := variableValues(snapshot, "tmp_table_size", "max_heap_table_size"); ok {
		tempTableRAM := snapshot.Variables["temptable_max_ram"]
		status := HealthOK
		if maxHeapSize < tmpTableSize || (tempTableRAM > 0 && tempTableRAM < 2*1024*1024*1024) {
			status = HealthWarning
		}
		checks = append(checks, HealthCheck{Name: "Temp table config", Display: fmt.Sprintf("tmp %.0fMB, heap %.0fMB, RAM %.0fMB", tmpTableSize/1024/1024, maxHeapSize/1024/1024, tempTableRAM/1024/1024), Status: status})
	}

	if uptime, written, ok := values(snapshot.Status, "Uptime", "Innodb_os_log_written"); ok && uptime > 0 && written > 0 {
		if capacity, capacityOK := snapshot.Variables["innodb_redo_log_capacity"]; capacityOK && capacity > 0 {
			minutes := (uptime / 60) * capacity / written
			status := HealthOK
			if minutes < 45 || minutes > 75 {
				status = HealthWarning
			}
			checks = append(checks, HealthCheck{Name: "InnoDB redo log", Value: minutes, Display: fmt.Sprintf("%.1f min", minutes), Status: status})
		}
	}

	return checks
}

func connectionCheck(name string, connections, maxConnections, usage float64) HealthCheck {
	status := HealthOK
	if usage > 80 {
		status = HealthCritical
	} else if usage > 70 {
		status = HealthWarning
	}
	return HealthCheck{Name: name, Value: usage, Display: fmt.Sprintf("%.0f / %.0f (%.1f%%)", connections, maxConnections, usage), Status: status}
}

func percentageCheck(name string, value, warning, critical float64) HealthCheck {
	status := HealthOK
	if value < critical {
		status = HealthCritical
	} else if value < warning {
		status = HealthWarning
	}
	return HealthCheck{Name: name, Value: value, Display: fmt.Sprintf("%.2f%%", value), Status: status}
}
