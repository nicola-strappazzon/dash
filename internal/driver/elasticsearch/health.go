package elasticsearch

import (
	"context"
	"encoding/json"
	"fmt"
)

type HealthResponse struct {
	ClusterName                 string  `json:"cluster_name"`
	Status                      string  `json:"status"`
	TimedOut                    bool    `json:"timed_out"`
	NumberOfNodes               int     `json:"number_of_nodes"`
	NumberOfDataNodes           int     `json:"number_of_data_nodes"`
	ActivePrimaryShards         int     `json:"active_primary_shards"`
	ActiveShards                int     `json:"active_shards"`
	RelocatingShards            int     `json:"relocating_shards"`
	InitializingShards          int     `json:"initializing_shards"`
	UnassignedShards            int     `json:"unassigned_shards"`
	DelayedUnassignedShards     int     `json:"delayed_unassigned_shards"`
	NumberOfPendingTasks        int     `json:"number_of_pending_tasks"`
	NumberOfInFlightFetch       int     `json:"number_of_in_flight_fetch"`
	TaskMaxWaitingInQueueMillis int     `json:"task_max_waiting_in_queue_millis"`
	ActiveShardsPercentAsNumber float64 `json:"active_shards_percent_as_number"`
}

// Health fetches _cluster/health.
func (e *Elasticsearch) Health(ctx context.Context) (HealthResponse, error) {
	var health HealthResponse

	res, err := e.client.Cluster.Health(
		e.client.Cluster.Health.WithContext(ctx),
	)
	if err != nil {
		return health, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return health, fmt.Errorf("unexpected response: %s", res.String())
	}

	if err := json.NewDecoder(res.Body).Decode(&health); err != nil {
		return health, fmt.Errorf("failed to decode _cluster/health: %w", err)
	}

	return health, nil
}
