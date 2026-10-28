package stats

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fatih/color"
	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/nicola-strappazzon/dash/internal/driver/elasticsearch"
)

func TestArguments(t *testing.T) {
	for _, args := range [][]string{{"example-index"}, {"one", "two"}} {
		cmd := NewCommand(context.Background(), &command.Options{})
		if err := cmd.Args(cmd, args); err == nil {
			t.Errorf("expected error for %v", args)
		}
	}
	cmd := NewCommand(context.Background(), &command.Options{})
	if err := cmd.Args(cmd, nil); err != nil {
		t.Errorf("no filter: %v", err)
	}
}

func TestIndexFilter(t *testing.T) {
	for _, target := range []string{"", "example-a", "example-*", "one,two", "alias"} {
		t.Run("index="+target, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				prefix := ""
				if target != "" {
					prefix = "/" + target
				}
				if r.URL.Path != prefix+"/_stats/docs,store,search,get,indexing" {
					t.Errorf("unexpected request: %s", r.URL)
				}
				w.Header().Set("X-Elastic-Product", "Elasticsearch")
				fmt.Fprint(w, `{"_shards":{"failed":0},"indices":{}}`)
			}))
			defer server.Close()
			opts := &command.Options{}
			opts.Elasticsearch.Address = server.URL
			captureOutput(t, func() {
				cmd := NewCommand(context.Background(), opts)
				args := []string{}
				if target != "" {
					args = []string{"--index", target}
				}
				cmd.SetArgs(args)
				if err := cmd.Execute(); err != nil {
					t.Fatal(err)
				}
			})
		})
	}
}

func captureOutput(t *testing.T, run func()) string {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "output")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	stdout := os.Stdout
	colorOutput := color.Output
	os.Stdout = file
	color.Output = file
	defer func() { os.Stdout = stdout; color.Output = colorOutput }()
	run()
	data, err := os.ReadFile(file.Name())
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestRender(t *testing.T) {
	before := elasticsearch.IndexStatistics{Name: "example-a", UUID: "id", SampledAt: time.Unix(100, 0),
		Activity: elasticsearch.IndexActivity{Searches: 10, SearchTimeMillis: 200, Gets: 5, Indexed: 30, Deleted: 1}}
	after := before
	after.SampledAt = before.SampledAt.Add(10 * time.Second)
	after.Activity = elasticsearch.IndexActivity{Searches: 30, SearchTimeMillis: 300, Gets: 15, Indexed: 60, Deleted: 6}
	reset := before
	reset.SampledAt = after.SampledAt
	reset.Activity = elasticsearch.IndexActivity{}
	for _, tc := range []struct {
		name     string
		watch    bool
		sample   elasticsearch.IndexStatistics
		previous map[string]elasticsearch.IndexStatistics
		want     string
	}{
		{"cumulative", false, after, nil, "30 10.00 15 60 6"},
		{"first watch sample", true, after, nil, "- - - - -"},
		{"watch rates", true, after, map[string]elasticsearch.IndexStatistics{before.Name: before}, "2.00 5.00 1.00 3.00 0.50"},
		{"counter reset", true, reset, map[string]elasticsearch.IndexStatistics{before.Name: before}, "- - - - -"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			output := captureOutput(t, func() {
				if err := render(context.Background(), []elasticsearch.IndexStatistics{tc.sample}, tc.previous, tc.watch); err != nil {
					t.Fatal(err)
				}
			})
			var row string
			for _, line := range strings.Split(output, "\n") {
				if strings.Contains(line, "example-a") {
					row = strings.Join(strings.Fields(line), " ")
				}
			}
			if !strings.HasSuffix(row, tc.want) {
				t.Errorf("row = %q, want suffix %q", row, tc.want)
			}
			if strings.Contains(output, "SEARCH/s") != tc.watch {
				t.Errorf("wrong headers: %s", output)
			}
		})
	}
}

func TestWatch(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/example-*/_stats/docs,store,search,get,indexing" {
			t.Errorf("unexpected request: %s", r.URL)
		}
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		if requests == 3 {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, `{"error":"monitor privilege revoked"}`)
			return
		}
		fmt.Fprintf(w, `{"_shards":{"failed":0},"indices":{"example-a":{"uuid":"id","primaries":{"indexing":{"index_total":%d}},"total":{"search":{"query_total":%d}},"shards":{}}}}`, requests, requests)
	}))
	defer server.Close()
	opts := &command.Options{Watch: time.Millisecond}
	opts.Elasticsearch.Address = server.URL
	output := captureOutput(t, func() {
		cmd := NewCommand(context.Background(), opts)
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
		cmd.SetArgs([]string{"--index", "example-*"})
		if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "403") {
			t.Fatalf("expected polling error, got %v", err)
		}
	})
	if requests != 3 || strings.Count(output, "example-a") != 2 {
		t.Errorf("requests = %d, output = %s", requests, output)
	}
	if !strings.Contains(output, "SEARCH/s") || !strings.Contains(output, "no comparable sample") {
		t.Errorf("missing watch output: %s", output)
	}
}
