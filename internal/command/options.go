package command

import "time"

type Options struct {
	Clear                bool
	Watch                time.Duration
	ESAddress            string
	ESUsername           string
	ESPassword           string
	ESInsecureTLS        bool
	ESSnapshotRepository string
}
