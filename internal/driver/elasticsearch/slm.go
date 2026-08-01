package elasticsearch

import (
	"context"
	"encoding/json"
	"fmt"
)

type SlmRetention struct {
	ExpireAfter string `json:"expire_after"`
	MinCount    int    `json:"min_count"`
	MaxCount    int    `json:"max_count"`
}

type SlmInvocation struct {
	SnapshotName string `json:"snapshot_name"`
	Time         any    `json:"time"`
}

type SlmLifecycleItem struct {
	Policy struct {
		Name       string        `json:"name"`
		Schedule   string        `json:"schedule"`
		Repository string        `json:"repository"`
		Retention  *SlmRetention `json:"retention"`
	} `json:"policy"`
	NextExecutionMillis int64          `json:"next_execution_millis"`
	LastSuccess         *SlmInvocation `json:"last_success"`
	LastFailure         *SlmInvocation `json:"last_failure"`
	Stats               struct {
		SnapshotsTaken           any `json:"snapshots_taken"`
		SnapshotsFailed          any `json:"snapshots_failed"`
		SnapshotsDeleted         any `json:"snapshots_deleted"`
		SnapshotDeletionFailures any `json:"snapshot_deletion_failures"`
	} `json:"stats"`
}

// SlmLifecycle fetches _slm/policy, keyed by policy id.
func (e *Elasticsearch) SlmLifecycle(ctx context.Context) (map[string]SlmLifecycleItem, error) {
	res, err := e.client.SlmGetLifecycle(
		e.client.SlmGetLifecycle.WithContext(ctx),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("unexpected response from _slm/policy: %s", res.String())
	}

	var lifecycle map[string]SlmLifecycleItem
	if err := json.NewDecoder(res.Body).Decode(&lifecycle); err != nil {
		return nil, fmt.Errorf("failed to decode _slm/policy: %w", err)
	}

	return lifecycle, nil
}

type SlmPolicyStat struct {
	SnapshotsTaken           int64
	SnapshotsFailed          int64
	SnapshotsDeleted         int64
	SnapshotDeletionFailures int64
}

// SlmStats fetches _slm/stats, keyed by policy id.
func (e *Elasticsearch) SlmStats(ctx context.Context) (map[string]SlmPolicyStat, error) {
	res, err := e.client.SlmGetStats(
		e.client.SlmGetStats.WithContext(ctx),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("unexpected response from _slm/stats: %s", res.String())
	}

	var stats struct {
		PolicyStats []struct {
			Policy                   string `json:"policy"`
			SnapshotsTaken           int64  `json:"snapshots_taken"`
			SnapshotsFailed          int64  `json:"snapshots_failed"`
			SnapshotsDeleted         int64  `json:"snapshots_deleted"`
			SnapshotDeletionFailures int64  `json:"snapshot_deletion_failures"`
		} `json:"policy_stats"`
	}
	if err := json.NewDecoder(res.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode _slm/stats: %w", err)
	}

	byPolicy := make(map[string]SlmPolicyStat, len(stats.PolicyStats))
	for _, stat := range stats.PolicyStats {
		byPolicy[stat.Policy] = SlmPolicyStat{
			SnapshotsTaken:           stat.SnapshotsTaken,
			SnapshotsFailed:          stat.SnapshotsFailed,
			SnapshotsDeleted:         stat.SnapshotsDeleted,
			SnapshotDeletionFailures: stat.SnapshotDeletionFailures,
		}
	}

	return byPolicy, nil
}
