package elasticsearch

import (
	"context"
	"encoding/json"
	"fmt"
)

type IndexTotals struct {
	IndexTotal  int64
	IndexTimeMs int64
	Refresh     int64
	RefreshMs   int64
	Flush       int64
	FlushMs     int64
}

type indexHotShardStat struct {
	Indexing *struct {
		IndexTotal        int64 `json:"index_total"`
		IndexTimeInMillis int64 `json:"index_time_in_millis"`
	} `json:"indexing"`
	Refresh *struct {
		Total             int64 `json:"total"`
		TotalTimeInMillis int64 `json:"total_time_in_millis"`
	} `json:"refresh"`
	Flush *struct {
		Total             int64 `json:"total"`
		TotalTimeInMillis int64 `json:"total_time_in_millis"`
	} `json:"flush"`
}

type indexHotNodeStatsResponse struct {
	Nodes map[string]struct {
		Name    string `json:"name"`
		Indices struct {
			// Each array element is an object keyed by shard number (e.g.
			// {"0": {...}}), not the stats object itself — the shard number
			// isn't needed here, so we just range over the single entry.
			Shards map[string][]map[string]indexHotShardStat `json:"shards"`
		} `json:"indices"`
	} `json:"nodes"`
}

// IndexStats fetches _nodes/stats/indices (indexing, refresh, flush,
// level=shards) for node and aggregates the counters per index, summing
// across every shard of that index hosted on this node. These are
// cumulative totals since the shard started, not a live rate.
func (e *Elasticsearch) IndexStats(ctx context.Context, node string) (map[string]IndexTotals, error) {
	res, err := e.client.Nodes.Stats(
		e.client.Nodes.Stats.WithContext(ctx),
		e.client.Nodes.Stats.WithNodeID(node),
		e.client.Nodes.Stats.WithMetric("indices"),
		e.client.Nodes.Stats.WithIndexMetric("indexing", "refresh", "flush"),
		e.client.Nodes.Stats.WithLevel("shards"),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("unexpected response from _nodes/stats: %s", res.String())
	}

	var parsed indexHotNodeStatsResponse
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("failed to decode _nodes/stats: %w", err)
	}

	if len(parsed.Nodes) == 0 {
		return nil, fmt.Errorf("_nodes/stats returned no data for node %q — check the name", node)
	}

	totals := map[string]IndexTotals{}
	for _, n := range parsed.Nodes {
		for index, shards := range n.Indices.Shards {
			var t IndexTotals
			for _, entry := range shards {
				for _, s := range entry { // entry is {"<shard number>": stats}, e.g. {"0": {...}}
					if s.Indexing != nil {
						t.IndexTotal += s.Indexing.IndexTotal
						t.IndexTimeMs += s.Indexing.IndexTimeInMillis
					}
					if s.Refresh != nil {
						t.Refresh += s.Refresh.Total
						t.RefreshMs += s.Refresh.TotalTimeInMillis
					}
					if s.Flush != nil {
						t.Flush += s.Flush.Total
						t.FlushMs += s.Flush.TotalTimeInMillis
					}
				}
			}
			totals[index] = t
		}
	}

	return totals, nil
}
