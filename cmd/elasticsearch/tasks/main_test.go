package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/nicola-strappazzon/dash/internal/command"
)

func TestDescribe(t *testing.T) {
	const body = `{"completed":true,"task":{"id":9007199254740993,"status":{"custom_metric":42}},"response":{"failures":[{"reason":"checksum failed"}]}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/_tasks/worker:123" {
			t.Errorf("unexpected endpoint: %s", r.URL.Path)
		}
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		fmt.Fprint(w, body)
	}))
	defer server.Close()
	output, err := os.CreateTemp(t.TempDir(), "output")
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()
	stdout := os.Stdout
	os.Stdout = output
	defer func() { os.Stdout = stdout }()
	opts := &command.Options{}
	opts.Elasticsearch.Address = server.URL
	cmd := NewCommand(context.Background(), opts)
	cmd.SetArgs([]string{"--describe=worker:123"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(output.Name())
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(data) || !strings.Contains(string(data), "\n  \"completed\": true") || !strings.Contains(string(data), "9007199254740993") || !strings.Contains(string(data), "custom_metric") || !strings.Contains(string(data), "checksum failed") {
		t.Errorf("missing or altered task details: %s", data)
	}
}

func TestDescribeFlags(t *testing.T) {
	for _, args := range [][]string{{"--describe="}, {"--describe=worker:123", "--subtasks"}} {
		cmd := NewCommand(context.Background(), &command.Options{})
		cmd.SetArgs(args)
		if err := cmd.Execute(); err == nil {
			t.Errorf("expected validation error for %v", args)
		}
	}
}

func TestSubtasks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		fmt.Fprint(w, `{"nodes":{"node":{"name":"worker","tasks":{
			"node:1":{"action":"root-action","running_time_in_nanos":3000000000},
			"node:2":{"action":"child-action","parent_task_id":"node:1","running_time_in_nanos":2000000000},
			"node:3":{"action":"unset-parent-action","parent_task_id":"-1","running_time_in_nanos":1000000000}
		}}}}`)
	}))
	defer server.Close()
	for _, subtasks := range []bool{false, true} {
		t.Run(fmt.Sprintf("subtasks=%v", subtasks), func(t *testing.T) {
			output, err := os.CreateTemp(t.TempDir(), "output")
			if err != nil {
				t.Fatal(err)
			}
			defer output.Close()
			stdout := os.Stdout
			os.Stdout = output
			defer func() { os.Stdout = stdout }()
			opts := &command.Options{}
			opts.Elasticsearch.Address = server.URL
			cmd := NewCommand(context.Background(), opts)
			args := []string{}
			if subtasks {
				args = append(args, "--subtasks")
			}
			cmd.SetArgs(args)
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(output.Name())
			if err != nil {
				t.Fatal(err)
			}
			text := string(data)
			for _, action := range []string{"root-action", "unset-parent-action"} {
				if !strings.Contains(text, action) {
					t.Errorf("missing %s in output: %s", action, text)
				}
			}
			if strings.Contains(text, "child-action") != subtasks {
				t.Errorf("unexpected child visibility: %s", text)
			}
			if strings.Contains(text, "1 subtasks hidden") == subtasks {
				t.Errorf("unexpected hidden count: %s", text)
			}
		})
	}
}
