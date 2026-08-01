package elasticsearch

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/elastic/go-elasticsearch/v8/esapi"
)

type Shard struct {
	Index  string `json:"index"`
	Shard  string `json:"shard"`
	Prirep string `json:"prirep"`
	State  string `json:"state"`
	Docs   string `json:"docs"`
	Store  string `json:"store"`
	IP     string `json:"ip"`
	Node   string `json:"node"`
}

// Shards fetches _cat/shards, sorted by sortBy (e.g. "state,index,shard" or
// "store:desc"). indexFilter is an optional index name or pattern (e.g.
// "cr-*"); pass "" to fetch every index.
func (e *Elasticsearch) Shards(ctx context.Context, sortBy, indexFilter string) ([]Shard, error) {
	requestOpts := []func(*esapi.CatShardsRequest){
		e.client.Cat.Shards.WithContext(ctx),
		e.client.Cat.Shards.WithFormat("json"),
		e.client.Cat.Shards.WithBytes("b"),
		e.client.Cat.Shards.WithS(sortBy),
		e.client.Cat.Shards.WithH("index", "shard", "prirep", "state", "docs", "store", "ip", "node"),
	}
	if indexFilter != "" {
		requestOpts = append(requestOpts, e.client.Cat.Shards.WithIndex(indexFilter))
	}

	res, err := e.client.Cat.Shards(requestOpts...)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("unexpected response from _cat/shards: %s", res.String())
	}

	var shards []Shard
	if err := json.NewDecoder(res.Body).Decode(&shards); err != nil {
		return nil, fmt.Errorf("failed to decode _cat/shards: %w", err)
	}

	return shards, nil
}
