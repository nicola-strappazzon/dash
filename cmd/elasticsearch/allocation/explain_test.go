package allocation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/nicola-strappazzon/dash/internal/command"
)

func TestExplainValidation(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{nil, "--index"},
		{[]string{"--index", "example-a"}, "shard"},
		{[]string{"--index", " ", "--shard", "0"}, "--index"},
		{[]string{"--index", "example-a", "--shard", "-1"}, "--shard"},
		{[]string{"--index", "example-a", "--shard", "no"}, "invalid"},
		{[]string{"unexpected"}, "unknown command"},
	} {
		cmd := newExplainCommand(context.Background(), &command.Options{})
		cmd.SilenceUsage, cmd.SilenceErrors = true, true
		cmd.SetArgs(tc.args)
		if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("args %v: got %v, want %q", tc.args, err, tc.want)
		}
	}
}

func TestExplainCommand(t *testing.T) {
	for _, primary := range []bool{false, true} {
		t.Run(fmt.Sprint(primary), func(t *testing.T) {
			// The complete reason must remain visible even on narrow terminals.
			reason := strings.Repeat("allocation detail ", 20) + "required.setting=1"
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/_cluster/allocation/explain" {
					t.Errorf("unexpected endpoint: %s", r.URL)
				}
				var body struct {
					Index   string `json:"index"`
					Shard   int    `json:"shard"`
					Primary bool   `json:"primary"`
					Node    string `json:"current_node"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if body.Index != "example-a" || body.Shard != 0 || body.Primary != primary || body.Node != "data-a" {
					t.Errorf("selector: %+v", body)
				}
				w.Header().Set("X-Elastic-Product", "Elasticsearch")
				fmt.Fprintf(w, `{"index":"example-a","shard":0,"primary":%v,"current_state":"started","current_node":{"name":"data-a"},"can_remain_on_current_node":"no","can_remain_decisions":[{"decider":"filter","decision":"NO","explanation":"excluded source"}],"can_move_to_other_node":"no","move_explanation":"No eligible target","node_allocation_decisions":[{"node_name":"data-b","node_decision":"throttled","deciders":[{"decider":"throttling","decision":"THROTTLE","explanation":%q}]}]}`, primary, reason)
			}))
			defer server.Close()
			output, err := os.CreateTemp(t.TempDir(), "output")
			if err != nil {
				t.Fatal(err)
			}
			defer output.Close()
			stdout, colorOutput := os.Stdout, color.Output
			os.Stdout, color.Output = output, output
			defer func() { os.Stdout, color.Output = stdout, colorOutput }()
			opts := &command.Options{}
			opts.Elasticsearch.Address = server.URL
			cmd := NewCommand(context.Background(), opts)
			args := []string{"explain", "--index", "example-a", "--shard", "0", "--node", "data-a"}
			if primary {
				args = append(args, "--primary")
			}
			cmd.SetArgs(args)
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(output.Name())
			if err != nil {
				t.Fatal(err)
			}
			copy := "replica"
			if primary {
				copy = "primary"
			}
			for _, want := range []string{"example-a", copy, "data-a", "data-b", "THROTTLED", "excluded source", "No eligible target", reason} {
				if !strings.Contains(string(data), want) {
					t.Errorf("missing %q in %s", want, data)
				}
			}
		})
	}
}
