package elasticsearch

import (
	"context"

	"github.com/nicola-strappazzon/dash/cmd/elasticsearch/all"
	"github.com/nicola-strappazzon/dash/cmd/elasticsearch/health"
	"github.com/nicola-strappazzon/dash/cmd/elasticsearch/indices"
	"github.com/nicola-strappazzon/dash/cmd/elasticsearch/nodes"
	"github.com/nicola-strappazzon/dash/cmd/elasticsearch/recovery"
	"github.com/nicola-strappazzon/dash/cmd/elasticsearch/shards"
	"github.com/nicola-strappazzon/dash/cmd/elasticsearch/slm"
	"github.com/nicola-strappazzon/dash/cmd/elasticsearch/snapshots"
	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/spf13/cobra"
)

func NewCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "elasticsearch [all|health|nodes|indices|shards|recovery|snapshots|slm...]",
		Short: "Elasticsearch dashboard",
		Args:  cobra.ArbitraryArgs,
		RunE: command.Repeat(ctx, opts, func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}

			return renderSections(ctx, opts, args)
		}),
	}
	cmd.PersistentFlags().StringVar(&opts.Elasticsearch.Address, "address", "", "Elasticsearch address, e.g. https://localhost:9200")
	cmd.PersistentFlags().StringVarP(&opts.Elasticsearch.Username, "username", "u", "", "Elasticsearch username")
	cmd.PersistentFlags().StringVarP(&opts.Elasticsearch.Password, "password", "p", "", "Elasticsearch password")
	cmd.PersistentFlags().BoolVar(&opts.Elasticsearch.InsecureTLS, "insecure-skip-verify", false, "skip Elasticsearch TLS certificate verification")
	cmd.PersistentFlags().StringVar(&opts.Elasticsearch.SnapshotRepository, "snapshot-repository", "snapshots", "Elasticsearch snapshot repository name")

	cmd.AddCommand(
		all.NewCommand(ctx, opts),
		health.NewCommand(ctx, opts),
		nodes.NewCommand(ctx, opts),
		indices.NewCommand(ctx, opts),
		shards.NewCommand(ctx, opts),
		recovery.NewCommand(ctx, opts),
		snapshots.NewCommand(ctx, opts),
		slm.NewCommand(ctx, opts),
	)

	return cmd
}
