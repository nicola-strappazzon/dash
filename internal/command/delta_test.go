package command

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/nicola-strappazzon/go-table"
)

func TestDeltaCountersAndGauges(t *testing.T) {
	var tracker deltaTracker
	for i, sample := range []struct {
		counter string
		gauge   int
		want    string
	}{
		{"9007199254740993", 5, "-"},
		{"9007199254741005", 3, "12"},
		{"9007199254741005", 8, "0"},
		{"2", 1, "-"},
		{"5", 2, "3"},
		{"NULL", 0, "-"},
		{"100", 6, "-"},
		{"101", 7, "1"},
	} {
		tracker.begin()
		tbl := NewTable()
		tbl.Title("Status")
		tbl.Add("counter", sample.counter)
		tbl.Add("gauge", sample.gauge)
		tbl.Add("unknown", 987654321)
		tbl.Counter(1, "counter")
		tracker.apply(tbl)
		rows := tbl.Rows()
		if got := fmt.Sprint(rows[0].Fields[1].Value); got != sample.want {
			t.Fatalf("sample %d: got %s; want %s", i, got, sample.want)
		}
		if rows[1].Fields[1].Value != sample.gauge || rows[2].Fields[1].Value != 987654321 {
			t.Fatalf("sample %d changed gauge or unknown metric", i)
		}
		tracker.end()
	}
}

func TestDeltaSeriesIdentity(t *testing.T) {
	var tracker deltaTracker
	for iteration := 0; iteration < 2; iteration++ {
		tracker.begin()
		for _, title := range []string{"write", "search"} {
			tbl := NewTable()
			tbl.Title(title)
			tbl.Identity(0, 1)
			tbl.Counter(2)
			if iteration == 0 {
				tbl.Add("node", "pool-a", 100)
				tbl.Add("node", "pool-b", 1000)
			} else {
				tbl.Add("node", "pool-b", 1007) // Reordered rows still match their series.
				tbl.Add("node", "pool-a", 103)
			}
			tracker.apply(tbl)
			if iteration == 1 {
				for i, want := range []string{"7", "3"} {
					if got := fmt.Sprint(tbl.Rows()[i].Fields[2].Value); got != want {
						t.Fatalf("%s: got %s; want %s", title, got, want)
					}
				}
			}
		}
		tracker.end()
	}
	// A series missing in a sample must establish a new baseline on return.
	tracker.begin()
	tracker.end()
	tracker.begin()
	tbl := NewTable()
	tbl.Title("write")
	tbl.Identity(0, 1)
	tbl.Counter(2)
	tbl.Add("node", "pool-a", 200)
	tracker.apply(tbl)
	if _, ok := tbl.Rows()[0].Fields[2].Value.(pendingDelta); !ok {
		t.Fatal("reappearing series reused a stale baseline")
	}
}

func TestPendingDeltaSerialization(t *testing.T) {
	var tracker deltaTracker
	tracker.begin()
	tbl := NewTable()
	tbl.Add("a", 10)
	tbl.Column(0, table.Column{Name: "NAME"})
	tbl.Column(1, table.Column{Name: "COUNT"})
	tbl.Counter(1)
	tracker.apply(tbl)
	data, err := tbl.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	var doc document
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	if value := doc.Rows[0]["COUNT"]; value != nil {
		t.Fatalf("pending delta serialized as %v", value)
	}
}
