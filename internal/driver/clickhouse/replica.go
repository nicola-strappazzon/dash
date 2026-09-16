package clickhouse

import "context"

type ReplicaInfo struct {
	Host              string `ch:"host"`
	ReplicatedTables  uint64 `ch:"replicated_tables"`
	MaxDelay          uint64 `ch:"max_delay"`
	MaxQueue          uint32 `ch:"max_queue"`
	MaxReplicationLag int64  `ch:"max_replication_lag"`
	ReadonlyReplicas  uint64 `ch:"readonly_replicas"`
	ExpiredSessions   uint64 `ch:"expired_sessions"`
}

// Replicas fetches per-host replication health across every replica of the
// given cluster.
func (ch *ClickHouse) Replicas(ctx context.Context, cluster string) ([]ReplicaInfo, error) {
	const query = `
SELECT
    hostName() AS host,
    count() AS replicated_tables,
    max(absolute_delay) AS max_delay,
    max(queue_size) AS max_queue,
    max(log_max_index - log_pointer) AS max_replication_lag,
    sum(is_readonly) AS readonly_replicas,
    sum(is_session_expired) AS expired_sessions
FROM clusterAllReplicas(?, system.replicas)
GROUP BY host
ORDER BY host`

	var replicas []ReplicaInfo
	if err := ch.conn.Select(ctx, &replicas, query, cluster); err != nil {
		return nil, err
	}

	return replicas, nil
}
