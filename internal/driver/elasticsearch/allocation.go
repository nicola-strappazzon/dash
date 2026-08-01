package elasticsearch

import (
	"context"
	"encoding/json"
	"fmt"
)

type Allocation struct {
	Node        string `json:"node"`
	Shards      string `json:"shards"`
	DiskIndices string `json:"disk.indices"`
	DiskUsed    string `json:"disk.used"`
	DiskAvail   string `json:"disk.avail"`
	DiskTotal   string `json:"disk.total"`
	DiskPercent string `json:"disk.percent"`
}

// Allocation fetches _cat/allocation, sorted by disk percent (fullest first).
func (e *Elasticsearch) Allocation(ctx context.Context) ([]Allocation, error) {
	res, err := e.client.Cat.Allocation(
		e.client.Cat.Allocation.WithContext(ctx),
		e.client.Cat.Allocation.WithFormat("json"),
		e.client.Cat.Allocation.WithBytes("b"),
		e.client.Cat.Allocation.WithS("disk.percent:desc"),
		e.client.Cat.Allocation.WithH("node", "shards", "disk.indices", "disk.used", "disk.avail", "disk.total", "disk.percent"),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("unexpected response from _cat/allocation: %s", res.String())
	}

	var allocations []Allocation
	if err := json.NewDecoder(res.Body).Decode(&allocations); err != nil {
		return nil, fmt.Errorf("failed to decode _cat/allocation: %w", err)
	}

	return allocations, nil
}
