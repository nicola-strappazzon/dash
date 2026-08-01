package elasticsearch

import (
	"context"
	"encoding/json"
	"fmt"
)

type ThreadPool struct {
	NodeName string `json:"node_name"`
	Name     string `json:"name"`
	Active   string `json:"active"`
	Queue    string `json:"queue"`
	Rejected string `json:"rejected"`
	Size     string `json:"size"`
}

// ThreadPools fetches _cat/thread_pool for the "write" pool.
func (e *Elasticsearch) ThreadPools(ctx context.Context) ([]ThreadPool, error) {
	res, err := e.client.Cat.ThreadPool(
		e.client.Cat.ThreadPool.WithContext(ctx),
		e.client.Cat.ThreadPool.WithThreadPoolPatterns("write"),
		e.client.Cat.ThreadPool.WithFormat("json"),
		e.client.Cat.ThreadPool.WithH("node_name", "name", "active", "queue", "rejected", "size"),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("unexpected response from _cat/thread_pool: %s", res.String())
	}

	var pools []ThreadPool
	if err := json.NewDecoder(res.Body).Decode(&pools); err != nil {
		return nil, fmt.Errorf("failed to decode _cat/thread_pool: %w", err)
	}

	return pools, nil
}
