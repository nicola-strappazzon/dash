package elasticsearch

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTaskDetail(t *testing.T) {
	for _, tc := range []struct {
		name      string
		code      int
		body      string
		wantError bool
	}{
		{"running", 200, `{"completed":false,"task":{"status":{"total":10}}}`, false},
		{"completed", 200, `{"completed":true,"response":{"created":10,"failures":[]}}`, false},
		{"failed task", 200, `{"completed":true,"error":{"reason":"failure"}}`, false},
		{"not found", 404, `{"error":{"type":"resource_not_found_exception"}}`, true},
		{"invalid JSON", 200, `{`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/_tasks/worker:123" {
					t.Errorf("unexpected endpoint: %s", r.URL.Path)
				}
				w.Header().Set("X-Elastic-Product", "Elasticsearch")
				w.WriteHeader(tc.code)
				fmt.Fprint(w, tc.body)
			}))
			defer server.Close()
			es, err := New(Config{Addresses: []string{server.URL}})
			if err != nil {
				t.Fatal(err)
			}
			defer es.Close()
			detail, err := es.TaskDetail(context.Background(), "worker:123")
			if (err != nil) != tc.wantError {
				t.Fatalf("error = %v, wantError = %v", err, tc.wantError)
			}
			if !tc.wantError && string(detail) != tc.body {
				t.Errorf("detail = %s, want %s", detail, tc.body)
			}
		})
	}
}

func TestTasks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/_tasks" || r.URL.Query().Get("detailed") != "true" || r.URL.Query().Get("group_by") != "nodes" {
			t.Errorf("unexpected request: %s", r.URL)
		}
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		fmt.Fprint(w, `{"nodes":{"node-a":{"name":"worker-a","tasks":{
			"node-a:2":{"action":"indices:data/write/reindex","running_time_in_nanos":2000000000,"status":{"total":100,"created":20,"updated":3}},
			"node-a:1":{"action":"cluster:monitor/tasks/lists","running_time_in_nanos":1}
		}},"node-b":{"name":"worker-b","tasks":{
			"node-b:3":{"action":"indices:data/read/search","running_time_in_nanos":2000000000}
		}}}}`)
	}))
	defer server.Close()
	es, err := New(Config{Addresses: []string{server.URL}})
	if err != nil {
		t.Fatal(err)
	}
	defer es.Close()
	tasks, err := es.Tasks(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 3 {
		t.Fatalf("got %d tasks, want 3", len(tasks))
	}
	for i, id := range []string{"node-a:2", "node-b:3", "node-a:1"} {
		if tasks[i].TaskID != id {
			t.Errorf("task %d: got %s, want %s", i, tasks[i].TaskID, id)
		}
	}
	if tasks[0].Node != "worker-a" || tasks[0].Status == nil || tasks[0].Status.Total == nil || *tasks[0].Status.Total != 100 || tasks[0].Status.Created != 20 || tasks[0].Status.Updated != 3 {
		t.Errorf("unexpected reindex task: %+v", tasks[0])
	}
	if tasks[1].Status != nil {
		t.Error("search task should have no progress status")
	}
}

func TestTasksResponses(t *testing.T) {
	for _, tc := range []struct {
		name      string
		code      int
		body      string
		wantError bool
	}{
		{"empty", 200, `{"nodes":{}}`, false},
		{"forbidden", 403, `{"error":"forbidden"}`, true},
		{"invalid JSON", 200, `{`, true},
		{"node failure", 200, `{"nodes":{},"node_failures":[{"reason":"unavailable"}]}`, true},
		{"task failure", 200, `{"nodes":{},"task_failures":[{"reason":"unavailable"}]}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Elastic-Product", "Elasticsearch")
				w.WriteHeader(tc.code)
				fmt.Fprint(w, tc.body)
			}))
			defer server.Close()
			es, err := New(Config{Addresses: []string{server.URL}})
			if err != nil {
				t.Fatal(err)
			}
			defer es.Close()
			tasks, err := es.Tasks(context.Background())
			if (err != nil) != tc.wantError {
				t.Fatalf("error = %v, wantError = %v", err, tc.wantError)
			}
			if len(tasks) != 0 {
				t.Errorf("unexpected tasks: %v", tasks)
			}
		})
	}
}
