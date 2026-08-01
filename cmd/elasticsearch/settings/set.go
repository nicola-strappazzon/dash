package settings

import (
	"context"

	"github.com/nicola-strappazzon/dash/cmd/elasticsearch/internal/runner"
	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/nicola-strappazzon/dash/internal/driver/elasticsearch"
	"github.com/spf13/cobra"
)

func newSetCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	var apply bool
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set an arbitrary cluster setting (persistent and transient)",
		Long: "PUTs the given key/value to both persistent and transient cluster settings. " +
			"Use this for anything not covered by exclude-node/include-node/exclude-index/" +
			"include-index, e.g. indices.recovery.max_concurrent_file_chunks.",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, value := args[0], args[1]
			return runner.WithElasticsearch(opts, func(es *elasticsearch.Elasticsearch) error {
				return putClusterSettings(ctx, es, map[string]any{key: value}, apply)
			})
		},
	}
	cmd.Flags().BoolVarP(&apply, "yes", "y", false, "apply the change (default is a dry run that only shows what would be sent)")
	return cmd
}

func newSetIndexCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	var apply bool
	cmd := &cobra.Command{
		Use:   "set-index <index-name-or-pattern> <key> <value>",
		Short: "Set an arbitrary index setting (supports wildcard patterns like cr-*)",
		Long: "PUTs the given key/value to the given index(es) via PUT <index>/_settings. The " +
			"index argument can be a wildcard pattern (e.g. cr-*) to apply it to every " +
			"matching index in one shot. Use this for anything not covered by exclude-index/" +
			"include-index, e.g. index.refresh_interval.",
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			index, key, value := args[0], args[1], args[2]
			return runner.WithElasticsearch(opts, func(es *elasticsearch.Elasticsearch) error {
				return putIndexSettings(ctx, es, index, map[string]any{key: value}, apply)
			})
		},
	}
	cmd.Flags().BoolVarP(&apply, "yes", "y", false, "apply the change (default is a dry run that only shows what would be sent)")
	return cmd
}
