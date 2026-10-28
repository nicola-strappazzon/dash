package command

import (
	"context"
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/nicola-strappazzon/go-table"
)

func TestJSONFlagValue(t *testing.T) {
	opts := &Options{Format: "table"}
	ctx := WithOutput(context.Background(), opts)
	if JSON(ctx) || JSON(context.Background()) {
		t.Fatal("JSON should default to false")
	}
	opts.Format = "json"
	if !JSON(ctx) {
		t.Fatal("context did not observe parsed flag")
	}
}

func TestPrintSerializationError(t *testing.T) {
	opts := &Options{Format: "json"}
	err := Print(WithOutput(context.Background(), opts), table.New().Add(math.NaN()))
	if _, ok := err.(*json.UnsupportedValueError); !ok {
		t.Fatalf("serialization error was not propagated: %v", err)
	}
}

func TestCSVOutput(t *testing.T) {
	now := time.Unix(1793188800, 0)
	for _, tt := range []struct {
		name    string
		options Options
		columns []string
		rows    []map[string]any
		want    string
	}{
		{"raw numbers", Options{Format: "csv"}, []string{"HOST", "BYTES", "LAG"}, []map[string]any{{"HOST": "node,a", "BYTES": json.Number("9007199254740993"), "LAG": 0}}, "HOST,BYTES,LAG\n\"node,a\",9007199254740993,0\n"},
		{"timestamp", Options{Format: "csv", Timestamp: true}, []string{"LAG"}, []map[string]any{{"LAG": 0}}, "timestamp,LAG\n1793188800,0\n"},
		{"metric with identity", Options{Format: "csv", Metric: " lag ", Timestamp: true}, []string{"HOST", "LAG"}, []map[string]any{{"HOST": "a", "LAG": 0}, {"HOST": "b", "LAG": 2}}, "1793188800,a,0\n1793188800,b,2\n"},
		{"named metric", Options{Format: "csv", Metric: "Replication lag", Timestamp: true}, []string{"METRIC", "VALUE"}, []map[string]any{{"METRIC": "Replication lag", "VALUE": 0}}, "1793188800,0\n"},
		{"check before status", Options{Metric: "Connections"}, []string{"CHECK", "VALUE", "STATUS"}, []map[string]any{{"CHECK": "Connections", "VALUE": 0, "STATUS": "OK"}}, "0\n"},
		{"plain scalar", Options{Metric: "lag"}, []string{"HOST", "LAG"}, []map[string]any{{"HOST": "a", "LAG": 0}}, "0\n"},
		{"null", Options{Format: "csv", Metric: "lag"}, []string{"LAG"}, []map[string]any{{"LAG": nil}}, "NULL\n"},
		{"empty table", Options{Format: "csv"}, []string{"HOST", "LAG"}, nil, "HOST,LAG\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			out := &output{options: &tt.options}
			var buf strings.Builder
			err := out.print(&buf, document{Rows: tt.rows}, tt.columns, now)
			if err != nil {
				t.Fatal(err)
			}
			if got := buf.String(); got != tt.want {
				t.Fatalf("got %q; want %q", got, tt.want)
			}
		})
	}
}

func TestCSVWatchHeader(t *testing.T) {
	out := &output{options: &Options{Format: "csv"}}
	var buf strings.Builder
	for i := 0; i < 2; i++ {
		if err := out.print(&buf, document{Rows: []map[string]any{{"LAG": i}}}, []string{"LAG"}, time.Time{}); err != nil {
			t.Fatal(err)
		}
	}
	if got, want := buf.String(), "LAG\n0\n1\n"; got != want {
		t.Fatalf("got %q; want %q", got, want)
	}
}

func TestMetricAcrossViews(t *testing.T) {
	out := &output{options: &Options{Format: "csv", Metric: "lag"}}
	var buf strings.Builder
	for _, column := range []string{"OTHER", "LAG"} {
		if err := out.print(&buf, document{Rows: []map[string]any{{column: 0}}}, []string{column}, time.Time{}); err != nil {
			t.Fatal(err)
		}
	}
	if !out.matched || buf.String() != "0\n" {
		t.Fatalf("matched %v, output %q", out.matched, buf.String())
	}
}

func TestTableColumns(t *testing.T) {
	tbl := NewTable()
	tbl.Column(0, table.Column{Name: "HOST"})
	tbl.Column(1, table.Column{Name: "LAG"})
	tbl.SortBy(0)
	if got := tbl.outputColumns(); !reflect.DeepEqual(got, []string{"HOST", "LAG"}) {
		t.Fatal(got)
	}
}
