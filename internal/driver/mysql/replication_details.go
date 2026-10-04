package mysql

import (
	"context"
	"fmt"
	"strconv"
)

type ReplicationStatus struct {
	SourceHost         string
	SourcePort         int64
	SourceLogFile      string
	ReadSourceLogPos   int64
	RelaySourceLogFile string
	ExecSourceLogPos   int64
	RelayLogFile       string
	RelayLogPos        int64
	RelayLogSpace      int64
	AutoPosition       bool
	RetrievedGTIDSet   string
	ExecutedGTIDSet    string
}

type ReplicationReport struct {
	Status ReplicationStatus
	Checks []HealthCheck
}

// ReplicationDetails returns the diagnostic state of a replica. isReplica is
// false when the server has no replication channel configured.
func (m *MySQL) ReplicationDetails(ctx context.Context) (ReplicationReport, bool, error) {
	status, isReplica, err := m.replicaStatus(ctx)
	if err != nil {
		return ReplicationReport{}, false, err
	}
	if !isReplica {
		return ReplicationReport{}, false, nil
	}
	return ReplicationReport{
		Status: replicationStatus(status),
		Checks: replicationChecks(status),
	}, true, nil
}

func replicationStatus(status map[string]string) ReplicationStatus {
	return ReplicationStatus{
		SourceHost:         firstStatus(status, "Source_Host", "Master_Host"),
		SourcePort:         statusInt(status, "Source_Port", "Master_Port"),
		SourceLogFile:      firstStatus(status, "Source_Log_File", "Master_Log_File"),
		ReadSourceLogPos:   statusInt(status, "Read_Source_Log_Pos", "Read_Master_Log_Pos"),
		RelaySourceLogFile: firstStatus(status, "Relay_Source_Log_File", "Relay_Master_Log_File"),
		ExecSourceLogPos:   statusInt(status, "Exec_Source_Log_Pos", "Exec_Master_Log_Pos"),
		RelayLogFile:       firstStatus(status, "Relay_Log_File"),
		RelayLogPos:        statusInt(status, "Relay_Log_Pos"),
		RelayLogSpace:      statusInt(status, "Relay_Log_Space"),
		AutoPosition:       statusInt(status, "Auto_Position") == 1,
		RetrievedGTIDSet:   firstStatus(status, "Retrieved_Gtid_Set"),
		ExecutedGTIDSet:    firstStatus(status, "Executed_Gtid_Set"),
	}
}

func replicationChecks(status map[string]string) []HealthCheck {
	checks := []HealthCheck{
		replicaIOCheck(firstStatus(status, "Replica_IO_Running", "Slave_IO_Running")),
		replicaSQLCheck(firstStatus(status, "Replica_SQL_Running", "Slave_SQL_Running")),
		replicationLagCheck(firstStatus(status, "Seconds_Behind_Source", "Seconds_Behind_Master")),
		replicationErrorCheck("Last I/O error", statusInt(status, "Last_IO_Errno"), firstStatus(status, "Last_IO_Error")),
		replicationErrorCheck("Last SQL error", statusInt(status, "Last_SQL_Errno"), firstStatus(status, "Last_SQL_Error")),
	}
	if check := gtidDriftCheck(replicationStatus(status)); check != nil {
		checks = append(checks, *check)
	}
	return checks
}

func replicaIOCheck(value string) HealthCheck {
	status := HealthCritical
	if value == "Yes" {
		status = HealthOK
	} else if value == "Connecting" {
		status = HealthWarning
	}
	return HealthCheck{Name: "Replica I/O thread", Display: value, Status: status}
}

func replicaSQLCheck(value string) HealthCheck {
	status := HealthCritical
	if value == "Yes" {
		status = HealthOK
	}
	return HealthCheck{Name: "Replica SQL thread", Display: value, Status: status}
}

func replicationLagCheck(value string) HealthCheck {
	if value == "" {
		return HealthCheck{Name: "Replication lag", Display: "NULL", Status: HealthCritical}
	}
	lag, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return HealthCheck{Name: "Replication lag", Display: value, Status: HealthCritical}
	}
	status := HealthOK
	if lag > 300 {
		status = HealthCritical
	} else if lag > 30 {
		status = HealthWarning
	}
	return HealthCheck{Name: "Replication lag", Value: lag, Display: fmt.Sprintf("%.0fs", lag), Status: status}
}

func replicationErrorCheck(name string, errno int64, message string) HealthCheck {
	if errno == 0 {
		return HealthCheck{Name: name, Display: "none", Status: HealthOK}
	}
	return HealthCheck{Name: name, Display: fmt.Sprintf("errno %d · %s", errno, message), Status: HealthCritical}
}

func gtidDriftCheck(status ReplicationStatus) *HealthCheck {
	if status.RetrievedGTIDSet == "" && status.ExecutedGTIDSet == "" {
		return nil
	}
	if status.RetrievedGTIDSet == status.ExecutedGTIDSet {
		return &HealthCheck{Name: "GTID drift", Display: "none", Status: HealthOK}
	}
	return &HealthCheck{Name: "GTID drift", Display: "relay log pending", Status: HealthWarning}
}

func statusInt(status map[string]string, names ...string) int64 {
	value, _ := strconv.ParseInt(firstStatus(status, names...), 10, 64)
	return value
}
