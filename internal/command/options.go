package command

import "time"

type Options struct {
	Clear         bool
	Watch         time.Duration
	Elasticsearch struct {
		Address            string
		Username           string
		Password           string
		InsecureTLS        bool
		SnapshotRepository string
	}
}
