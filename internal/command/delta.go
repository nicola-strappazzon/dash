package command

import (
	"encoding/json"
	"fmt"
	"math/big"
	"slices"
)

type counterKey struct {
	view   string
	row    string
	column int
}

// deltaTracker keeps only consecutive samples so reappearing series establish
// a fresh baseline. Integer arithmetic preserves large database counters.
type deltaTracker struct {
	previous map[counterKey]*big.Int
	current  map[counterKey]*big.Int
}

func (d *deltaTracker) begin() { d.current = make(map[counterKey]*big.Int) }
func (d *deltaTracker) end()   { d.previous = d.current }

func (d *deltaTracker) apply(tbl *Table) {
	identity := tbl.identity
	if len(identity) == 0 {
		identity = []int{0}
	}
	for _, row := range tbl.Rows() {
		ids := make([]any, len(identity))
		for i, column := range identity {
			ids[i] = row.Fields[column].Value
		}
		data, _ := json.Marshal(ids)
		for column, labels := range tbl.counters {
			if len(labels) > 0 && !slices.Contains(labels, fmt.Sprint(row.Fields[0].Value)) {
				continue
			}
			key := counterKey{tbl.title, string(data), column}
			current, ok := new(big.Int).SetString(fmt.Sprint(row.Fields[column].Value), 10)
			previous := d.previous[key]
			row.Fields[column].Value = pendingDelta{}
			if !ok || current.Sign() < 0 {
				continue
			}
			d.current[key] = current
			if previous == nil || current.Cmp(previous) < 0 {
				continue
			}
			delta := new(big.Int).Sub(current, previous)
			if delta.IsInt64() {
				row.Fields[column].Value = delta.Int64()
			} else {
				row.Fields[column].Value = json.Number(delta.String())
			}
		}
	}
}

// A counter without a comparable sample has no delta, rather than zero.
type pendingDelta struct{}

func (pendingDelta) String() string               { return "-" }
func (pendingDelta) MarshalJSON() ([]byte, error) { return []byte("null"), nil }
