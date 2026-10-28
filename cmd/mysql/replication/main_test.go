package replication

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/nicola-strappazzon/dash/internal/command"
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
			got, err := metricValue(report, tt.name)
			if err != nil || got != tt.want {
				t.Fatalf("got %q, %v; want %q", got, err, tt.want)
			}
		})
	}
	for _, name := range []string{"unknown", "lag", "Retrieved GTID"} {
		if _, err := metricValue(report, name); err == nil {
			t.Errorf("expected error for %q", name)
		}
	}
}

func TestMetricFlag(t *testing.T) {
	for _, tt := range []struct {
		value     string
		args      []string
		wantError bool
	}{
		{"Replication lag", nil, false},
		{"", nil, true},
		{"  ", nil, true},
		{"Source", []string{"health"}, true},
	} {
		cmd := NewCommand(context.Background(), &command.Options{})
		if err := cmd.Flags().Set("metric", tt.value); err != nil {
			t.Fatal(err)
		}
		err := cmd.PreRunE(cmd, tt.args)
		if (err != nil) != tt.wantError {
			t.Fatalf("metric %q, args %v: %v", tt.value, tt.args, err)
		}
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
		got, err := metricValue(report, "Replication lag")
		if err != nil || got != tt.want {
			t.Fatalf("got %q, %v; want %q", got, err, tt.want)
		}
		if rows := reportRows(report); rows[0][1] != tt.display {
			t.Fatalf("table value changed: %q", rows[0][1])
		}
	}
}

func TestTimestampFlag(t *testing.T) {
	for _, metric := range []string{"", "Replication lag"} {
		cmd := NewCommand(context.Background(), &command.Options{})
		if err := cmd.Flags().Set("timestamp", "true"); err != nil {
			t.Fatal(err)
		}
		if metric != "" {
			if err := cmd.Flags().Set("metric", metric); err != nil {
				t.Fatal(err)
			}
		}
		err := cmd.PreRunE(cmd, nil)
		if (err != nil) != (metric == "") {
			t.Fatalf("metric %q: %v", metric, err)
		}
	}
}

func TestPrintMetric(t *testing.T) {
	for _, tt := range []struct {
		timestamp   bool
		value, want string
	}{
		{false, "0", "0\n"},
		{true, "0", "1793188800,0\n"},
		{true, "1.25", "1793188800,1.25\n"},
		{true, "NULL", "1793188800,NULL\n"},
	} {
		var out strings.Builder
		if err := printMetric(&out, tt.value, tt.timestamp, time.Unix(1793188800, 0)); err != nil {
			t.Fatal(err)
		}
		if got := out.String(); got != tt.want {
			t.Fatalf("got %q; want %q", got, tt.want)
		}
	}
}
