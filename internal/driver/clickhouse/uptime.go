package clickhouse

import "context"

type UptimeInfo struct {
	Host          string `ch:"host"`
	UptimeSeconds uint32 `ch:"uptime_seconds"`
	Version       string `ch:"version"`
}

// Uptime fetches the hostname, uptime and version of every replica in the
// given cluster.
func (ch *ClickHouse) Uptime(ctx context.Context, cluster string) ([]UptimeInfo, error) {
	const query = `
SELECT
    hostName() AS host,
    uptime() AS uptime_seconds,
    version() AS version
FROM clusterAllReplicas(?, system.one)`

	var uptime []UptimeInfo
	if err := ch.conn.Select(ctx, &uptime, query, cluster); err != nil {
		return nil, err
	}

	return uptime, nil
}
