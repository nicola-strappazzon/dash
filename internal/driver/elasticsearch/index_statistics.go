package elasticsearch

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// IndexStatistics is a sample of index storage and cumulative activity counters.
type IndexStatistics struct {
	Name         string
	UUID         string
	SampledAt    time.Time
	Documents    int64
	PrimaryBytes int64
	TotalBytes   int64
	Activity     IndexActivity
	copies       map[string]IndexActivity
}

type IndexActivity struct {
	Searches         int64
	SearchTimeMillis int64
	Gets             int64
	Indexed          int64
	Deleted          int64
}

type IndexRates struct {
	Searches     float64
	SearchMillis float64
	Gets         float64
	Indexed      float64
	Deleted      float64
}

func (a IndexActivity) decreased(before IndexActivity) bool {
	return a.Searches < before.Searches || a.SearchTimeMillis < before.SearchTimeMillis ||
		a.Gets < before.Gets || a.Indexed < before.Indexed || a.Deleted < before.Deleted
}

// RatesSince rejects samples spanning index replacement, shard movement or counter resets.
func (s IndexStatistics) RatesSince(before IndexStatistics) (IndexRates, bool) {
	seconds := s.SampledAt.Sub(before.SampledAt).Seconds()
	if s.UUID != before.UUID || seconds <= 0 || len(s.copies) != len(before.copies) || s.Activity.decreased(before.Activity) {
		return IndexRates{}, false
	}
	for id, activity := range s.copies {
		previous, ok := before.copies[id]
		if !ok || activity.decreased(previous) {
			return IndexRates{}, false
		}
	}
	a, b := s.Activity, before.Activity
	rates := IndexRates{
		Searches: float64(a.Searches-b.Searches) / seconds,
		Gets:     float64(a.Gets-b.Gets) / seconds,
		Indexed:  float64(a.Indexed-b.Indexed) / seconds,
		Deleted:  float64(a.Deleted-b.Deleted) / seconds,
	}
	if a.Searches > b.Searches {
		rates.SearchMillis = float64(a.SearchTimeMillis-b.SearchTimeMillis) / float64(a.Searches-b.Searches)
	}
	return rates, true
}

// indexStatisticsMetrics is used for primary totals, all-copy totals and individual copies.
type indexStatisticsMetrics struct {
	Docs struct {
		Count int64 `json:"count"`
	} `json:"docs"`
	Store struct {
		Bytes int64 `json:"size_in_bytes"`
	} `json:"store"`
	Indexing struct {
		Indexed int64 `json:"index_total"`
		Deleted int64 `json:"delete_total"`
	} `json:"indexing"`
	Search struct {
		Queries    int64 `json:"query_total"`
		TimeMillis int64 `json:"query_time_in_millis"`
	} `json:"search"`
	Get struct {
		Total int64 `json:"total"`
	} `json:"get"`
}

func (m indexStatisticsMetrics) activity() IndexActivity {
	return IndexActivity{Searches: m.Search.Queries, SearchTimeMillis: m.Search.TimeMillis,
		Gets: m.Get.Total, Indexed: m.Indexing.Indexed, Deleted: m.Indexing.Deleted}
}

func (e *Elasticsearch) IndexStatistics(ctx context.Context, target string) ([]IndexStatistics, error) {
	var indicesFilter []string
	if target != "" {
		indicesFilter = strings.Split(target, ",")
	}
	res, err := e.client.Indices.Stats(
		e.client.Indices.Stats.WithContext(ctx),
		e.client.Indices.Stats.WithIndex(indicesFilter...),
		e.client.Indices.Stats.WithMetric("docs", "store", "search", "get", "indexing"),
		e.client.Indices.Stats.WithLevel("shards"),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.IsError() {
		return nil, fmt.Errorf("unexpected response from index stats: %s", res.String())
	}
	var response struct {
		Shards struct {
			Failed int `json:"failed"`
		} `json:"_shards"`
		Indices map[string]struct {
			UUID      string                 `json:"uuid"`
			Primaries indexStatisticsMetrics `json:"primaries"`
			Total     indexStatisticsMetrics `json:"total"`
			Shards    map[string][]struct {
				Routing struct {
					State          string `json:"state"`
					Primary        bool   `json:"primary"`
					Node           string `json:"node"`
					RelocatingNode string `json:"relocating_node"`
				} `json:"routing"`
				indexStatisticsMetrics
			} `json:"shards"`
		} `json:"indices"`
	}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode index stats: %w", err)
	}
	if response.Shards.Failed > 0 {
		return nil, fmt.Errorf("incomplete index stats: %d shards failed", response.Shards.Failed)
	}
	sampledAt := time.Now()
	indices := make([]IndexStatistics, 0, len(response.Indices))
	for name, index := range response.Indices {
		activity := index.Total.activity()
		activity.Indexed = index.Primaries.Indexing.Indexed
		activity.Deleted = index.Primaries.Indexing.Deleted
		sample := IndexStatistics{Name: name, UUID: index.UUID, SampledAt: sampledAt,
			Documents: index.Primaries.Docs.Count, PrimaryBytes: index.Primaries.Store.Bytes,
			TotalBytes: index.Total.Store.Bytes, Activity: activity, copies: make(map[string]IndexActivity)}
		for shard, copies := range index.Shards {
			for _, copy := range copies {
				r := copy.Routing
				id := fmt.Sprintf("%s/%s/%t/%s/%s", shard, r.Node, r.Primary, r.State, r.RelocatingNode)
				sample.copies[id] = copy.activity()
			}
		}
		indices = append(indices, sample)
	}
	sort.Slice(indices, func(i, j int) bool { return indices[i].Name < indices[j].Name })
	return indices, nil
}
