package snapshotscmd

import (
	"fmt"

	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/nicola-strappazzon/dash/internal/dashboard"
)

func withElasticsearch(opts *command.Options, run func(*dashboard.Elasticsearch) error) error {
	if opts.ESAddress == "" {
		return fmt.Errorf("missing Elasticsearch address: pass --address")
	}

	es, err := dashboard.NewElasticsearch(dashboard.Config{
		Addresses:          []string{opts.ESAddress},
		Username:           opts.ESUsername,
		Password:           opts.ESPassword,
		InsecureSkipVerify: opts.ESInsecureTLS,
	})
	if err != nil {
		return err
	}

	return run(es)
}
