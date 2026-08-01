package elasticsearchcmd

import (
	"context"

	snapshotscmd "github.com/nicola-strappazzon/dash/cmd/elasticsearch/snapshots"
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
	cmd.PersistentFlags().StringVar(&opts.ESAddress, "address", "", "Elasticsearch address, e.g. https://localhost:9200")
	cmd.PersistentFlags().StringVarP(&opts.ESUsername, "username", "u", "", "Elasticsearch username")
	cmd.PersistentFlags().StringVarP(&opts.ESPassword, "password", "p", "", "Elasticsearch password")
	cmd.PersistentFlags().BoolVar(&opts.ESInsecureTLS, "insecure-skip-verify", false, "skip Elasticsearch TLS certificate verification")
	cmd.PersistentFlags().StringVar(&opts.ESSnapshotRepository, "snapshot-repository", "snapshots", "Elasticsearch snapshot repository name")

	cmd.AddCommand(
		newAllCommand(ctx, opts),
		newHealthCommand(ctx, opts),
		newNodesCommand(ctx, opts),
		newIndicesCommand(ctx, opts),
		newShardsCommand(ctx, opts),
		newRecoveryCommand(ctx, opts),
		snapshotscmd.NewCommand(ctx, opts),
		newSLMCommand(ctx, opts),
	)

	return cmd
}
