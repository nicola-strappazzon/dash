package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/fatih/color"
)

func captureCommand(t *testing.T, args []string) (string, string, error) {
	t.Helper()
	out, err := os.CreateTemp(t.TempDir(), "stdout")
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	errout, err := os.CreateTemp(t.TempDir(), "stderr")
	if err != nil {
		t.Fatal(err)
	}
	defer errout.Close()
	stdout, stderr, colorOutput := os.Stdout, os.Stderr, color.Output
	os.Stdout, os.Stderr, color.Output = out, errout, out
	defer func() { os.Stdout, os.Stderr, color.Output = stdout, stderr, colorOutput }()
	cmd := NewRootCommand(context.Background())
	cmd.SetArgs(args)
	runErr := cmd.Execute()
	data, err := os.ReadFile(out.Name())
	if err != nil {
		t.Fatal(err)
	}
	diagnostics, err := os.ReadFile(errout.Name())
	if err != nil {
		t.Fatal(err)
	}
	return string(data), string(diagnostics), runErr
}

func jsonDocuments(t *testing.T, text string) []map[string]json.RawMessage {
	t.Helper()
	var documents []map[string]json.RawMessage
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		var document map[string]json.RawMessage
		if err := json.Unmarshal(scanner.Bytes(), &document); err != nil {
			t.Fatalf("non-JSON stdout: %q: %v", scanner.Text(), err)
		}
		documents = append(documents, document)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return documents
}

func TestGlobalJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		switch r.URL.Path {
		case "/_cluster/health":
			fmt.Fprint(w, `{"cluster_name":"example","status":"green","number_of_nodes":3}`)
		case "/_cat/allocation":
			fmt.Fprint(w, `[{"node":"a-very-long-node-name-that-must-not-be-truncated","shards":"5","disk.indices":"1024","disk.used":"2048","disk.avail":"4096","disk.total":"6144","disk.percent":"33"}]`)
		case "/example-a/_stats/docs,store,search,get,indexing":
			fmt.Fprint(w, `{"_shards":{"failed":0},"indices":{"example-a":{"uuid":"id","primaries":{"docs":{"count":9007199254740993},"store":{"size_in_bytes":1024},"indexing":{"index_total":6}},"total":{"store":{"size_in_bytes":2048},"search":{"query_total":2,"query_time_in_millis":3},"get":{"total":4}},"shards":{}}}}`)
		case "/missing-*/_stats/docs,store,search,get,indexing":
			fmt.Fprint(w, `{"_shards":{"failed":0},"indices":{}}`)
		case "/_tasks":
			fmt.Fprint(w, `{"nodes":{"node":{"name":"data-a","tasks":{"node:1":{"action":"reindex","running_time_in_nanos":1000000000},"node:2":{"action":"search","parent_task_id":"node:1"}}}}}`)
		case "/_tasks/node:1":
			fmt.Fprint(w, `{"completed":true,"response":{"created":10,"failures":[]}}`)
		case "/_cluster/allocation/explain":
			fmt.Fprint(w, `{"index":"example-a","shard":0,"primary":false,"current_state":"unassigned","can_allocate":"throttled","allocate_explanation":"waiting for recovery slots"}`)
		case "/_nodes/hot_threads":
			fmt.Fprint(w, "::: node-a\n Hot threads at example\n cpu usage by thread example\n")
		default:
			t.Errorf("unexpected endpoint: %s", r.URL)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	for _, tc := range []struct {
		name      string
		args      []string
		documents int
		want      string
		stderr    string
	}{
		{"vertical health", []string{"health"}, 1, `"number_of_nodes":3`, ""},
		{"allocation", []string{"allocation"}, 1, `"DISK.AVAIL":4096`, ""},
		{"multiple sections", []string{"health", "allocation"}, 2, "must-not-be-truncated", ""},
		{"stats numeric values", []string{"stats", "--index", "example-a"}, 1, `"SEARCH.ms":1.5`, ""},
		{"stats no matches", []string{"stats", "--index", "missing-*"}, 1, `"rows":[]`, "No open indices matched"},
		{"tasks", []string{"tasks"}, 1, `"ACTION":"reindex"`, "1 subtasks hidden"},
		{"task detail", []string{"tasks", "--describe=node:1"}, 1, `"completed":true`, ""},
		{"explanation", []string{"allocation", "explain", "--index", "example-a", "--shard", "0"}, 1, `"can_allocate":"throttled"`, ""},
		{"hot threads", []string{"thread", "hot"}, 1, `"text":`, ""},
		{"settings dry run", []string{"settings", "set", "cluster.routing.allocation.enable", "all"}, 1, `"dry_run":true`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"--json", "elasticsearch", "--address", server.URL}, tc.args...)
			stdout, stderr, err := captureCommand(t, args)
			if err != nil {
				t.Fatal(err)
			}
			if got := len(jsonDocuments(t, stdout)); got != tc.documents {
				t.Errorf("got %d documents, want %d", got, tc.documents)
			}
			if !strings.Contains(stdout, tc.want) || strings.Contains(stdout, "\x1b") {
				t.Errorf("unexpected stdout: %s", stdout)
			}
			if tc.stderr != "" && !strings.Contains(stderr, tc.stderr) {
				t.Errorf("missing diagnostic in stderr: %s", stderr)
			}
			if tc.name == "stats numeric values" && !strings.Contains(stdout, "9007199254740993") {
				t.Errorf("integer precision lost: %s", stdout)
			}
		})
	}
	stdout, _, err := captureCommand(t, []string{"version", "--json"})
	if err != nil || len(jsonDocuments(t, stdout)) != 1 || !strings.Contains(stdout, `"version"`) {
		t.Fatalf("version JSON: %s, %v", stdout, err)
	}
}

func TestJSONWatchIgnoresClear(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		requests++
		if requests == 3 {
			w.WriteHeader(403)
			fmt.Fprint(w, `{"error":"monitor privilege revoked"}`)
			return
		}
		fmt.Fprintf(w, `{"_shards":{"failed":0},"indices":{"example-a":{"uuid":"same","primaries":{"indexing":{"index_total":%d}},"total":{"search":{"query_total":%d}},"shards":{}}}}`, requests, requests)
	}))
	defer server.Close()
	out, diagnostics, err := captureCommand(t, []string{"elasticsearch", "--address", server.URL, "stats", "--json", "--watch", "1ms", "--clear"})
	if err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("expected polling error, got %v", err)
	}
	if len(jsonDocuments(t, out)) != 2 || strings.Contains(out, "\x1b") {
		t.Errorf("invalid watch stream: %s", out)
	}
	if !strings.Contains(diagnostics, "no comparable sample") {
		t.Errorf("missing diagnostic: %s", diagnostics)
	}
}
