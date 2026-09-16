package clickhouse

import "context"

type DiskInfo struct {
	Host        string  `ch:"host"`
	Disk        string  `ch:"disk"`
	Path        string  `ch:"path"`
	Free        uint64  `ch:"free"`
	Total       uint64  `ch:"total"`
	UsedPercent float64 `ch:"used_percent"`
}

// Disks fetches per-disk free/total space of every replica in the given
// cluster.
func (ch *ClickHouse) Disks(ctx context.Context, cluster string) ([]DiskInfo, error) {
	const query = `
SELECT
    hostName() AS host,
    name AS disk,
    path,
    free_space AS free,
    total_space AS total,
    round(100 * (1 - free_space / total_space), 2) AS used_percent
FROM clusterAllReplicas(?, system.disks)
ORDER BY host, disk`

	var disks []DiskInfo
	if err := ch.conn.Select(ctx, &disks, query, cluster); err != nil {
		return nil, err
	}

	return disks, nil
}
