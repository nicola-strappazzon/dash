package command

import "time"

type Options struct {
	Format        string
	Metric        string
	Timestamp     bool
	Delta         bool
	Clear         bool
	Watch         time.Duration
	ClickHouse    ClickHouseOptions
	Elasticsearch ElasticsearchOptions
	MySQL         MySQLOptions
}
