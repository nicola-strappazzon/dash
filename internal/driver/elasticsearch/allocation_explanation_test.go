package elasticsearch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExplainAllocation(t *testing.T) {
	for _, tc := range []struct {
		name    string
		primary bool
		node    string
	}{
		{"replica", false, ""},
		{"primary on node", true, "data-a"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" || r.URL.Path != "/_cluster/allocation/explain" || r.URL.Query().Get("include_yes_decisions") != "false" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
				}
				var body map[string]json.RawMessage
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if string(body["index"]) != `"example-a"` || string(body["shard"]) != "0" || string(body["primary"]) != fmt.Sprint(tc.primary) {
					t.Errorf("unexpected selector: %s", body)
				}
				if tc.node == "" {
					if _, exists := body["current_node"]; exists {
						t.Error("current_node should be omitted")
					}
				} else if string(body["current_node"]) != `"data-a"` {
					t.Errorf("current_node = %s", body["current_node"])
				}
				w.Header().Set("X-Elastic-Product", "Elasticsearch")
				fmt.Fprintf(w, `{"index":"example-a","shard":0,"primary":%v,"current_state":"unassigned", "unassigned_info":{"reason":"INDEX_CREATED","at":"2026-01-01T00:00:00Z"},"can_allocate":"throttled","allocate_explanation":"Waiting for recovery slots","node_allocation_decisions":[{"node_name":"data-z","node_decision":"throttled","deciders":[{"decider":"throttling","decision":"THROTTLE","explanation":"incoming limit reached"}]},{"node_name":"data-a","node_decision":"yes"}]}`, tc.primary)
			}))
			defer server.Close()
			es, err := New(Config{Addresses: []string{server.URL}})
			if err != nil {
				t.Fatal(err)
			}
			defer es.Close()
			e, err := es.ExplainAllocation(context.Background(), "example-a", 0, tc.primary, tc.node)
			if err != nil {
				t.Fatal(err)
			}
			if e.Index != "example-a" || e.Shard != 0 || e.Primary != tc.primary || e.CanAllocate != "throttled" || e.UnassignedInfo.Reason != "INDEX_CREATED" {
				t.Errorf("unexpected explanation: %+v", e)
			}
			if len(e.NodeDecisions) != 2 || e.NodeDecisions[0].Name != "data-a" || e.NodeDecisions[1].Deciders[0].Explanation != "incoming limit reached" {
				t.Errorf("unexpected node decisions: %+v", e.NodeDecisions)
			}
		})
	}
}

func TestExplainAllocationResponses(t *testing.T) {
	for _, tc := range []struct {
		name      string
		code      int
		body      string
		wantError bool
	}{
		{"assigned", 200, `{"current_state":"started","current_node":{"name":"data-a"},"can_remain_on_current_node":"no","can_remain_decisions":[{"decider":"filter","decision":"NO","explanation":"excluded node"}],"can_move_to_other_node":"no","move_explanation":"No eligible target"}`, false},
		{"no valid copy", 200, `{"current_state":"unassigned","can_allocate":"no_valid_shard_copy","allocate_explanation":"No good copies"}`, false},
		{"relocating", 200, `{"current_state":"relocating","explanation":"Wait for relocation"}`, false},
		{"invalid selector", 400, `{"error":"shard not found"}`, true},
		{"missing privilege", 403, `{"error":"forbidden"}`, true},
		{"invalid JSON", 200, `{`, true},
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
			e, err := es.ExplainAllocation(context.Background(), "example-a", 0, true, "")
			if (err != nil) != tc.wantError {
				t.Fatalf("error = %v, wantError = %v", err, tc.wantError)
			}
			if tc.name == "assigned" && (e.CurrentNode.Name != "data-a" || e.CanRemainDecisions[0].Decider != "filter" || e.MoveExplanation != "No eligible target") {
				t.Errorf("assigned explanation lost fields: %+v", e)
			}
			if tc.name == "relocating" && e.Explanation != "Wait for relocation" {
				t.Errorf("relocation explanation lost: %+v", e)
			}
		})
	}
}
