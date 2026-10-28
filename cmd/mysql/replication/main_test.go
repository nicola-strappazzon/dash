package replication

import (
	"strings"
	"testing"

	"github.com/nicola-strappazzon/dash/internal/driver/mysql"
)

func TestMetricValue(t *testing.T) {
	report := mysql.ReplicationReport{
		Checks: []mysql.HealthCheck{{Name: "Replication lag", Display: "12s"}},
		Status: mysql.ReplicationStatus{SecondsBehindSource: "12", RelayLogSpace: 1048576, SourceHost: "source.example", SourcePort: 3306, AutoPosition: true, ExecutedGTIDSet: strings.Repeat("a", 200)},
	}
	for _, tt := range []struct{ name, want string }{
		{"Replication lag", "12"},
		{" replication LAG ", "12"},
		{"Source", "source.example:3306"},
		{"Auto position", "1"},
		{"Relay log space", "1048576"},
		{"Executed GTID", strings.Repeat("a", 200)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			for _, row := range reportRows(report, true) {
				if strings.EqualFold(row[0], strings.TrimSpace(tt.name)) {
					got = row[1]
				}
			}
			if got != tt.want {
				t.Fatalf("got %q; want %q", got, tt.want)
			}
		})
	}
}

func TestRawReplicationLag(t *testing.T) {
	for _, tt := range []struct{ raw, display, want string }{
		{"0", "0s", "0"},
		{"1.25", "1s", "1.25"},
		{"", "NULL", "NULL"},
	} {
		report := mysql.ReplicationReport{
			Status: mysql.ReplicationStatus{SecondsBehindSource: tt.raw},
			Checks: []mysql.HealthCheck{{Name: "Replication lag", Display: tt.display}},
		}
		got := reportRows(report, true)[0][1]
		if got != tt.want {
			t.Fatalf("got %q; want %q", got, tt.want)
		}
		if rows := reportRows(report, false); rows[0][1] != tt.display {
			t.Fatalf("table value changed: %q", rows[0][1])
		}
	}
}
