package all

import (
	"context"

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
	"github.com/spf13/cobra"
)

func NewCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	return runner.SectionCommand(ctx, opts, "all", "Show all Elasticsearch dashboard sections", func(ctx context.Context, es *elasticsearchdriver.Elasticsearch) error {
		return Render(ctx, es, opts.Elasticsearch.SnapshotRepository)
	})
}

func Render(ctx context.Context, es *elasticsearchdriver.Elasticsearch, snapshotRepository string) error {
	for _, render := range []runner.RenderFunc{
		health.Render,
		nodes.Render,
		indices.Render,
		shards.Render,
		recovery.Render,
		snapshots.Repository(snapshotRepository),
		slm.Render,
	} {
		if err := render(ctx, es); err != nil {
			return err
		}
	}

	return nil
}
