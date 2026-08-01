package elasticsearch

import (
	"context"
	"encoding/json"
	"fmt"
)

type SnapshotRepository struct {
	Type     string         `json:"type"`
	Settings map[string]any `json:"settings"`
}

// SnapshotRepositories fetches _snapshot (every registered repository).
func (e *Elasticsearch) SnapshotRepositories(ctx context.Context) (map[string]SnapshotRepository, error) {
	res, err := e.client.Snapshot.GetRepository(
		e.client.Snapshot.GetRepository.WithContext(ctx),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("unexpected response from _snapshot: %s", res.String())
	}

	var repositories map[string]SnapshotRepository
	if err := json.NewDecoder(res.Body).Decode(&repositories); err != nil {
		return nil, fmt.Errorf("failed to decode _snapshot: %w", err)
	}

	return repositories, nil
}

type Snapshot struct {
	ID               string `json:"id"`
	Status           string `json:"status"`
	StartEpoch       string `json:"start_epoch"`
	StartTime        string `json:"start_time"`
	EndEpoch         string `json:"end_epoch"`
	EndTime          string `json:"end_time"`
	Duration         string `json:"duration"`
	Indices          string `json:"indices"`
	SuccessfulShards string `json:"successful_shards"`
	FailedShards     string `json:"failed_shards"`
	TotalShards      string `json:"total_shards"`
}

// Snapshots fetches _cat/snapshots for the given repository, sorted by start time.
func (e *Elasticsearch) Snapshots(ctx context.Context, repository string) ([]Snapshot, error) {
	res, err := e.client.Cat.Snapshots(
		e.client.Cat.Snapshots.WithContext(ctx),
		e.client.Cat.Snapshots.WithRepository(repository),
		e.client.Cat.Snapshots.WithFormat("json"),
		e.client.Cat.Snapshots.WithS("start_epoch"),
		e.client.Cat.Snapshots.WithH("id", "status", "start_time", "end_time", "duration", "indices", "successful_shards", "failed_shards", "total_shards"),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("unexpected response from _cat/snapshots: %s", res.String())
	}

	var snapshots []Snapshot
	if err := json.NewDecoder(res.Body).Decode(&snapshots); err != nil {
		return nil, fmt.Errorf("failed to decode _cat/snapshots: %w", err)
	}

	return snapshots, nil
}
