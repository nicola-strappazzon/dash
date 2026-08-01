package elasticsearch

import (
	"context"
	"encoding/json"
	"fmt"
)

type Node struct {
	ID              string `json:"id"`
	IP              string `json:"ip"`
	Name            string `json:"name"`
	Master          string `json:"master"`
	NodeRole        string `json:"node.role"`
	HeapPercent     string `json:"heap.percent"`
	RAMPercent      string `json:"ram.percent"`
	CPU             string `json:"cpu"`
	Load1m          string `json:"load_1m"`
	DiskUsedPercent string `json:"disk.used_percent"`
	Uptime          string `json:"uptime"`
}

// Nodes fetches _cat/nodes.
func (e *Elasticsearch) Nodes(ctx context.Context) ([]Node, error) {
	res, err := e.client.Cat.Nodes(
		e.client.Cat.Nodes.WithContext(ctx),
		e.client.Cat.Nodes.WithFormat("json"),
		e.client.Cat.Nodes.WithH("id", "ip", "name", "master", "node.role", "heap.percent", "ram.percent", "cpu", "load_1m", "disk.used_percent", "uptime"),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("unexpected response from _cat/nodes: %s", res.String())
	}

	var nodes []Node
	if err := json.NewDecoder(res.Body).Decode(&nodes); err != nil {
		return nil, fmt.Errorf("failed to decode _cat/nodes: %w", err)
	}

	return nodes, nil
}
