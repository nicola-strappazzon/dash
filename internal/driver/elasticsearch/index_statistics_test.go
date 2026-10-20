package elasticsearch

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIndexStatistics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/example-*,other/_stats/docs,store,search,get,indexing" || r.URL.Query().Get("level") != "shards" {
			t.Errorf("unexpected request: %s", r.URL)
		}
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		fmt.Fprint(w, `{"_shards":{"failed":0},"indices":{
		"other":{"uuid":"other-id","primaries":{"docs":{"count":1}},"shards":{}},
		"example-a":{"uuid":"example-id",
		"primaries":{"docs":{"count":12},"store":{"size_in_bytes":1024},"indexing":{"index_total":40,"delete_total":3}},
		"total":{"store":{"size_in_bytes":2048},"indexing":{"index_total":80,"delete_total":6},"search":{"query_total":10,"query_time_in_millis":250},"get":{"total":4}},
		"shards":{"0":[
		{"routing":{"state":"STARTED","primary":true,"node":"node-a"},"indexing":{"index_total":40},"search":{"query_total":6}},
		{"routing":{"state":"STARTED","primary":false,"node":"node-b"},"indexing":{"index_total":40},"search":{"query_total":4}}
		]}}}}`)
	}))
	defer server.Close()
	es, err := New(Config{Addresses: []string{server.URL}})
	if err != nil {
		t.Fatal(err)
	}
	defer es.Close()
	indices, err := es.IndexStatistics(context.Background(), "example-*,other")
	if err != nil {
		t.Fatal(err)
	}
	if len(indices) != 2 || indices[0].Name != "example-a" || indices[1].Name != "other" {
		t.Fatalf("unexpected indices: %+v", indices)
	}
	s := indices[0]
	if s.Documents != 12 || s.PrimaryBytes != 1024 || s.TotalBytes != 2048 || s.UUID != "example-id" || s.SampledAt.IsZero() {
		t.Errorf("unexpected sample: %+v", s)
	}
	if s.Activity != (IndexActivity{Searches: 10, SearchTimeMillis: 250, Gets: 4, Indexed: 40, Deleted: 3}) {
		t.Errorf("wrong primary/all-copy aggregation: %+v", s.Activity)
	}
	if len(s.copies) != 2 || s.copies["0/node-a/true/STARTED/"].Searches != 6 || s.copies["0/node-b/false/STARTED/"].Searches != 4 {
		t.Errorf("missing per-copy counters: %+v", s.copies)
	}
}

func TestIndexStatisticsErrors(t *testing.T) {
	for _, tc := range []struct {
		name      string
		code      int
		body      string
		wantError bool
	}{
		{"empty", 200, `{"_shards":{"failed":0},"indices":{}}`, false},
		{"partial failure", 200, `{"_shards":{"failed":1},"indices":{}}`, true},
		{"missing index", 404, `{"error":"index_not_found_exception"}`, true},
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
			_, err = es.IndexStatistics(context.Background(), "example-*")
			if (err != nil) != tc.wantError {
				t.Fatalf("error = %v, wantError = %v", err, tc.wantError)
			}
		})
	}
}

func TestIndexStatisticsRates(t *testing.T) {
	start := time.Unix(100, 0)
	before := IndexStatistics{UUID: "same", SampledAt: start,
		Activity: IndexActivity{Searches: 100, SearchTimeMillis: 200, Gets: 20, Indexed: 50, Deleted: 10},
		copies:   map[string]IndexActivity{"copy-a": {Searches: 50}, "copy-b": {Searches: 50}}}
	after := IndexStatistics{UUID: "same", SampledAt: start.Add(10 * time.Second),
		Activity: IndexActivity{Searches: 120, SearchTimeMillis: 300, Gets: 30, Indexed: 80, Deleted: 15},
		copies:   map[string]IndexActivity{"copy-a": {Searches: 60}, "copy-b": {Searches: 60}}}
	rates, valid := after.RatesSince(before)
	if !valid || rates != (IndexRates{Searches: 2, SearchMillis: 5, Gets: 1, Indexed: 3, Deleted: .5}) {
		t.Fatalf("rates = %+v, valid = %v", rates, valid)
	}
	for _, tc := range []struct {
		name   string
		change func(*IndexStatistics)
	}{
		{"index recreated", func(s *IndexStatistics) { s.UUID = "replacement" }},
		{"no elapsed time", func(s *IndexStatistics) { s.SampledAt = start }},
		{"aggregate reset", func(s *IndexStatistics) { s.Activity.Gets = 0 }},
		{"copy removed", func(s *IndexStatistics) { s.copies = map[string]IndexActivity{"copy-a": {Searches: 120}} }},
		{"assignment changed", func(s *IndexStatistics) {
			s.copies = map[string]IndexActivity{"copy-a": {Searches: 60}, "new-copy": {Searches: 60}}
		}},
		{"reset masked by another copy", func(s *IndexStatistics) {
			s.copies = map[string]IndexActivity{"copy-a": {Searches: 10}, "copy-b": {Searches: 110}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sample := after
			tc.change(&sample)
			if _, valid := sample.RatesSince(before); valid {
				t.Error("expected a fresh baseline")
			}
		})
	}
	idle := before
	idle.SampledAt = after.SampledAt
	if rates, valid := idle.RatesSince(before); !valid || rates != (IndexRates{}) {
		t.Errorf("idle rates = %+v, valid = %v", rates, valid)
	}
}
