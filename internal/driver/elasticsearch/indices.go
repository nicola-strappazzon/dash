package elasticsearch

import (
	"context"
	"encoding/json"
	"fmt"
)

type Index struct {
	Health       string `json:"health"`
	Status       string `json:"status"`
	Name         string `json:"index"`
	Primary      string `json:"pri"`
	Replica      string `json:"rep"`
	DocsCount    string `json:"docs.count"`
	DocsDeleted  string `json:"docs.deleted"`
	StoreSize    string `json:"store.size"`
	PriStoreSize string `json:"pri.store.size"`
}

// Indices fetches _cat/indices, sorted by store size (biggest first).
func (e *Elasticsearch) Indices(ctx context.Context) ([]Index, error) {
	res, err := e.client.Cat.Indices(
		e.client.Cat.Indices.WithContext(ctx),
		e.client.Cat.Indices.WithFormat("json"),
		e.client.Cat.Indices.WithBytes("b"),
		e.client.Cat.Indices.WithS("store.size:desc"),
		e.client.Cat.Indices.WithH("health", "status", "index", "pri", "rep", "docs.count", "docs.deleted", "store.size", "pri.store.size"),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("unexpected response from _cat/indices: %s", res.String())
	}

	var indices []Index
	if err := json.NewDecoder(res.Body).Decode(&indices); err != nil {
		return nil, fmt.Errorf("failed to decode _cat/indices: %w", err)
	}

	return indices, nil
}
