package command

import "time"

type Options struct {
	Clear      bool
	Watch      time.Duration
	ClickHouse struct {
		Host        string
		Username    string
		Password    string
		TLS         bool
		InsecureTLS bool
		ClusterName string
	}
	Elasticsearch struct {
		Address            string
		Username           string
		Password           string
		InsecureTLS        bool
		SnapshotRepository string
	}
}
