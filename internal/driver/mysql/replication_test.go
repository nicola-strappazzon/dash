package mysql

import "testing"

func TestEvaluateReplication(t *testing.T) {
	checks := evaluateReplication(map[string]string{
		"Replica_IO_Running":    "Yes",
		"Replica_SQL_Running":   "Yes",
		"Seconds_Behind_Source": "45",
	})
	if len(checks) != 2 {
		t.Fatalf("got %d checks, want 2", len(checks))
	}
	if checks[0].Status != HealthOK {
		t.Errorf("thread status = %q, want %q", checks[0].Status, HealthOK)
	}
	if checks[1].Status != HealthWarning || checks[1].Display != "45s" {
		t.Errorf("lag check = %#v", checks[1])
	}
}

func TestEvaluateReplicationRejectsNullLag(t *testing.T) {
	checks := evaluateReplication(map[string]string{
		"Replica_IO_Running":  "Yes",
		"Replica_SQL_Running": "No",
	})
	if checks[0].Status != HealthCritical {
		t.Errorf("thread status = %q, want %q", checks[0].Status, HealthCritical)
	}
	if checks[1].Status != HealthCritical || checks[1].Display != "NULL" {
		t.Errorf("lag check = %#v", checks[1])
	}
}
