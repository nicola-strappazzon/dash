package cmd

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestOutputValidation(t *testing.T) {
	for _, tt := range []struct {
		args []string
		want string
	}{
		{[]string{"version", "--format", "xml"}, "invalid --format"},
		{[]string{"version", "--metric", " "}, "--metric requires"},
		{[]string{"version", "--format", "table", "--timestamp"}, "--timestamp requires"},
		{[]string{"version", "--format", "json", "--metric", "version", "--timestamp"}, "--timestamp requires"},
		{[]string{"version", "--metric", "missing"}, "unknown metric"},
		{[]string{"version", "--json"}, "unknown flag"},
		{[]string{"version", "--delta"}, "--delta requires --watch"},
		{[]string{"version", "--delta", "--watch", "0s"}, "--delta requires --watch"},
		{[]string{"version", "--metric", ", ,"}, "--metric requires"},
	} {
		out, _, err := captureCommand(t, tt.args)
		if err == nil || !strings.Contains(err.Error(), tt.want) || out != "" {
			t.Errorf("args %v: output %q, error %v", tt.args, out, err)
		}
	}
}

func TestGlobalCSVAndMetric(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		switch r.URL.Path {
		case "/_cat/allocation":
			fmt.Fprint(w, `[{"node":"node,a","shards":"5","disk.indices":"1024","disk.used":"2048","disk.avail":"4096","disk.total":"6144","disk.percent":"33"}]`)
		case "/_cluster/health":
			fmt.Fprint(w, `{"cluster_name":"example","status":"green","number_of_nodes":3}`)
		case "/_cluster/allocation/explain":
			fmt.Fprint(w, `{"index":"example-a","shard":0,"primary":false,"current_state":"unassigned","can_allocate":"throttled"}`)
		case "/_tasks/node:1":
			fmt.Fprint(w, `{"completed":true,"response":{"created":9007199254740993}}`)
		case "/_nodes/hot_threads":
			fmt.Fprint(w, "::: node-a\n cpu usage by thread sample\n")
		default:
			t.Errorf("unexpected endpoint: %s", r.URL)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	for _, tt := range []struct {
		name string
		args []string
		want string
	}{
		{"csv raw values", []string{"allocation", "--format", "csv"}, "NODE,SHARDS,DISK.INDICES,DISK.USED,DISK.AVAIL,DISK.TOTAL,DISK%\n\"node,a\",5,1024,2048,4096,6144,33\n"},
		{"plain metric", []string{"allocation", "--metric", "disk.avail"}, "4096\n"},
		{"csv metric", []string{"allocation", "--format", "csv", "--metric", "DISK.AVAIL"}, "\"node,a\",4096\n"},
		{"vertical metric", []string{"health", "--metric", "number_of_nodes"}, "3\n"},
		{"multiple columns", []string{"allocation", "--format", "csv", "--metric", "SHARDS,DISK.AVAIL"}, "\"node,a\",5,4096\n"},
		{"multiple vertical fields", []string{"health", "--metric", "number_of_nodes,Name"}, "3,example\n"},
		{"multiple sections", []string{"health", "allocation", "--metric", "DISK.AVAIL"}, "4096\n"},
		{"nested metric", []string{"tasks", "--describe", "node:1", "--metric", "response.created"}, "9007199254740993\n"},
		{"explain metric", []string{"allocation", "explain", "--index", "example-a", "--shard", "0", "--metric", "can_allocate"}, "throttled\n"},
		{"settings dry run", []string{"settings", "set", "cluster.routing.allocation.enable", "all", "--metric", "dry_run"}, "true\n"},
		{"hot threads CSV", []string{"thread", "hot", "--format", "csv"}, "text,title\n\"::: node-a\n cpu usage by thread sample\n\",Hot threads\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"elasticsearch", "--address", server.URL}, tt.args...)
			out, _, err := captureCommand(t, args)
			if err != nil {
				t.Fatal(err)
			}
			if out != tt.want {
				t.Fatalf("got %q; want %q", out, tt.want)
			}
		})
	}
	before := time.Now().Unix()
	out, _, err := captureCommand(t, []string{"elasticsearch", "--address", server.URL, "health", "--metric", "number_of_nodes", "--format", "csv", "--timestamp"})
	if err != nil {
		t.Fatal(err)
	}
	records, err := csv.NewReader(strings.NewReader(out)).ReadAll()
	if err != nil || len(records) != 1 || len(records[0]) != 3 {
		t.Fatalf("timestamp output: %q, %v", out, err)
	}
	stamp, err := strconv.ParseInt(records[0][0], 10, 64)
	if err != nil || stamp < before || stamp > time.Now().Unix() {
		t.Fatalf("invalid timestamp: %q", out)
	}
	if records[0][2] != "3" {
		t.Fatalf("invalid metric: %q", out)
	}
}

func TestCSVWatchIgnoresClear(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		requests++
		if requests == 3 {
			w.WriteHeader(403)
			fmt.Fprint(w, `{"error":"forbidden"}`)
			return
		}
		fmt.Fprintf(w, `{"cluster_name":"example","status":"green","number_of_nodes":%d}`, requests)
	}))
	defer server.Close()
	out, _, err := captureCommand(t, []string{"elasticsearch", "--address", server.URL, "health", "--format", "csv", "--watch", "1ms", "--clear"})
	if err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("expected polling error: %v", err)
	}
	records, err := csv.NewReader(strings.NewReader(out)).ReadAll()
	if err != nil || len(records) != 3 || strings.Contains(out, "\x1b") {
		t.Fatalf("invalid CSV stream %q: %v", out, err)
	}
	for i, name := range records[0] {
		if name == "number_of_nodes" && (records[1][i] != "1" || records[2][i] != "2") {
			t.Fatalf("invalid samples: %q", out)
		}
	}
}

func TestGlobalOutputFlagsInherited(t *testing.T) {
	root := NewRootCommand(context.Background())
	var visit func(*cobra.Command)
	visit = func(cmd *cobra.Command) {
		for _, child := range cmd.Commands() {
			for _, flag := range []string{"format", "metric", "timestamp", "delta"} {
				if child.InheritedFlags().Lookup(flag) == nil {
					t.Errorf("%s lacks --%s", child.CommandPath(), flag)
				}
			}
			if child.Flags().Lookup("json") != nil || child.InheritedFlags().Lookup("json") != nil {
				t.Errorf("%s still exposes --json", child.CommandPath())
			}
			visit(child)
		}
	}
	visit(root)
}

func TestDeltaWatchFormats(t *testing.T) {
	for _, format := range []string{"table", "json", "csv"} {
		t.Run(format, func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Elastic-Product", "Elasticsearch")
				requests++
				if requests == 4 {
					w.WriteHeader(403)
					fmt.Fprint(w, `{"error":"forbidden"}`)
					return
				}
				// The first cumulative value is deliberately large. ACTIVE is a gauge.
				fmt.Fprintf(w, `[{"node_name":"data-a","name":"write","active":"%d","queue":"0","rejected":"%d","size":"4"}]`, 4-requests, 1000000+requests*3)
			}))
			defer server.Close()
			args := []string{"elasticsearch", "--address", server.URL, "thread", "show", "--delta", "--watch", "1ms", "--format", format}
			out, _, err := captureCommand(t, args)
			if err == nil || !strings.Contains(err.Error(), "403") {
				t.Fatalf("expected polling error, got %v", err)
			}
			if strings.Contains(out, "1000003") || strings.Contains(out, "1000006") {
				t.Fatalf("printed a cumulative counter: %s", out)
			}
			switch format {
			case "csv":
				records, err := csv.NewReader(strings.NewReader(out)).ReadAll()
				if err != nil || len(records) != 3 {
					t.Fatalf("first sample was not skipped: %q, %v", out, err)
				}
				if records[1][2] != "2" || records[2][2] != "1" || records[1][4] != "3" || records[2][4] != "3" {
					t.Fatalf("incorrect gauges/deltas: %q", out)
				}
			case "json":
				docs := jsonDocuments(t, out)
				if len(docs) != 2 {
					t.Fatalf("first sample was not skipped: %s", out)
				}
				for i, doc := range docs {
					var rows []map[string]any
					if err := json.Unmarshal(doc["rows"], &rows); err != nil {
						t.Fatal(err)
					}
					if rows[0]["ACTIVE"] != float64(2-i) || rows[0]["REJECTED"] != float64(3) {
						t.Fatalf("incorrect gauges/deltas: %s", out)
					}
				}
			case "table":
				var rows []string
				for _, line := range strings.Split(out, "\n") {
					if strings.Contains(line, "data-a") {
						rows = append(rows, strings.Join(strings.Fields(line), " "))
					}
				}
				if len(rows) != 2 || rows[0] != "data-a write 2 0 3 4" || rows[1] != "data-a write 1 0 3 4" {
					t.Fatalf("incorrect table samples: %q", rows)
				}
			}
		})
	}
}
