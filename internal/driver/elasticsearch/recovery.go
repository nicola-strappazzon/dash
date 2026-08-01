package elasticsearch

import (
	"context"
	"encoding/json"
	"fmt"
)

type Recovery struct {
	Index                string `json:"index"`
	Shard                string `json:"shard"`
	Time                 string `json:"time"`
	Type                 string `json:"type"`
	Stage                string `json:"stage"`
	SourceNode           string `json:"source_node"`
	TargetNode           string `json:"target_node"`
	FilesRecovered       string `json:"files_recovered"`
	FilesTotal           string `json:"files_total"`
	FilesPercent         string `json:"files_percent"`
	BytesRecovered       string `json:"bytes_recovered"`
	BytesTotal           string `json:"bytes_total"`
	BytesPercent         string `json:"bytes_percent"`
	TranslogOpsRecovered string `json:"translog_ops_recovered"`
	TranslogOps          string `json:"translog_ops"`
	TranslogOpsPercent   string `json:"translog_ops_percent"`
}

// Recovery fetches _cat/recovery for active recoveries only.
func (e *Elasticsearch) Recovery(ctx context.Context) ([]Recovery, error) {
	res, err := e.client.Cat.Recovery(
		e.client.Cat.Recovery.WithContext(ctx),
		e.client.Cat.Recovery.WithFormat("json"),
		e.client.Cat.Recovery.WithActiveOnly(true),
		e.client.Cat.Recovery.WithBytes("b"),
		e.client.Cat.Recovery.WithTime("s"),
		e.client.Cat.Recovery.WithS("index,shard"),
		e.client.Cat.Recovery.WithH(
			"index",
			"shard",
			"time",
			"type",
			"stage",
			"source_node",
			"target_node",
			"files_recovered",
			"files_total",
			"files_percent",
			"bytes_recovered",
			"bytes_total",
			"bytes_percent",
			"translog_ops_recovered",
			"translog_ops",
			"translog_ops_percent",
		),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("unexpected response from _cat/recovery: %s", res.String())
	}

	var recoveries []Recovery
	if err := json.NewDecoder(res.Body).Decode(&recoveries); err != nil {
		return nil, fmt.Errorf("failed to decode _cat/recovery: %w", err)
	}

	return recoveries, nil
}
