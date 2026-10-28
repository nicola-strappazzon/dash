package command

import "time"

type Options struct {
	JSON          bool
	Clear         bool
	Watch         time.Duration
	ClickHouse    ClickHouseOptions
	Elasticsearch ElasticsearchOptions
	MySQL         MySQLOptions
}
