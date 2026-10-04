package command

import "time"

type Options struct {
	Clear         bool
	Watch         time.Duration
	ClickHouse    ClickHouseOptions
	Elasticsearch ElasticsearchOptions
	MySQL         MySQLOptions
}
