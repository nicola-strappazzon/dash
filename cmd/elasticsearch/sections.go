package elasticsearch

import (
	"context"
	"fmt"

	"github.com/nicola-strappazzon/dash/cmd/elasticsearch/all"
	"github.com/nicola-strappazzon/dash/cmd/elasticsearch/health"
	"github.com/nicola-strappazzon/dash/cmd/elasticsearch/indices"
	"github.com/nicola-strappazzon/dash/cmd/elasticsearch/internal/runner"
	"github.com/nicola-strappazzon/dash/cmd/elasticsearch/nodes"
	"github.com/nicola-strappazzon/dash/cmd/elasticsearch/recovery"
	"github.com/nicola-strappazzon/dash/cmd/elasticsearch/shards"
	"github.com/nicola-strappazzon/dash/cmd/elasticsearch/slm"
	"github.com/nicola-strappazzon/dash/cmd/elasticsearch/snapshots"
	"github.com/nicola-strappazzon/dash/internal/command"
	elasticsearchdriver "github.com/nicola-strappazzon/dash/internal/driver/elasticsearch"
)

func renderSections(ctx context.Context, opts *command.Options, sections []string) error {
	if err := validateSections(sections); err != nil {
		return err
	}

	return runner.WithElasticsearch(opts, func(es *elasticsearchdriver.Elasticsearch) error {
		for _, section := range sections {
			switch section {
			case "all":
				if err := all.Render(ctx, es, opts.Elasticsearch.SnapshotRepository); err != nil {
					return err
				}
			case "health":
				if err := health.Render(ctx, es); err != nil {
					return err
				}
			case "nodes":
				if err := nodes.Render(ctx, es); err != nil {
					return err
				}
			case "indices":
				if err := indices.Render(ctx, es); err != nil {
					return err
				}
			case "shards":
				if err := shards.Render(ctx, es); err != nil {
					return err
				}
			case "recovery":
				if err := recovery.Render(ctx, es); err != nil {
					return err
				}
			case "snapshots":
				if err := snapshots.Render(ctx, es, opts.Elasticsearch.SnapshotRepository); err != nil {
					return err
				}
			case "slm":
				if err := slm.Render(ctx, es); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unknown elasticsearch section %q", section)
			}
		}

		return nil
	})
}

func validateSections(sections []string) error {
	seen := map[string]bool{}
	for _, section := range sections {
		if section == "all" && len(sections) > 1 {
			return fmt.Errorf("all cannot be combined with other elasticsearch sections")
		}
		if seen[section] {
			return fmt.Errorf("duplicate elasticsearch section %q", section)
		}
		seen[section] = true
	}

	return nil
}
