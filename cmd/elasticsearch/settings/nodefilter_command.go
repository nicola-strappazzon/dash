package settings

import (
	"context"

	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/spf13/cobra"
)

func newExcludeNodeCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	var apply bool
	cmd := &cobra.Command{
		Use:   "exclude-node <node-name>",
		Short: "Stop allocating shards to a node (cluster-wide)",
		Long:  "Sets cluster.routing.allocation.exclude._name so Elasticsearch moves shards off the given node and stops allocating new ones to it. Use this to drain a node before maintenance or removal.\n\nTo move a single index off a node instead of draining it entirely, use exclude-index.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return setNodeFilter(ctx, opts, "exclude._name", args[0], "", apply)
		},
	}
	cmd.Flags().BoolVarP(&apply, "yes", "y", false, "apply the change (default is a dry run that only shows what would be sent)")
	return cmd
}

func newIncludeNodeCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	var apply bool
	cmd := &cobra.Command{
		Use:   "include-node <node-name>",
		Short: "Restrict shard allocation to a node (cluster-wide)",
		Long:  "Sets cluster.routing.allocation.include._name so Elasticsearch only allocates shards to the given node(s). Rarely needed; exclude-node is the common case for draining a node.\n\nTo restrict a single index to a node instead of the whole cluster, use include-index.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return setNodeFilter(ctx, opts, "include._name", args[0], "", apply)
		},
	}
	cmd.Flags().BoolVarP(&apply, "yes", "y", false, "apply the change (default is a dry run that only shows what would be sent)")
	return cmd
}

func newExcludeIndexCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	var apply bool
	var node string
	cmd := &cobra.Command{
		Use:   "exclude-index <index-name> --node <node-name>",
		Short: "Stop allocating a single index's shards to a node",
		Long:  "Sets index.routing.allocation.exclude._name on the given index, so Elasticsearch moves that index's shards off the given node and stops allocating new ones to it, without touching anything else on that node. Use this to relieve disk pressure by moving your biggest indices elsewhere; see `shards --node <node-name>` to find them.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return setNodeFilter(ctx, opts, "exclude._name", node, args[0], apply)
		},
	}
	cmd.Flags().BoolVarP(&apply, "yes", "y", false, "apply the change (default is a dry run that only shows what would be sent)")
	cmd.Flags().StringVar(&node, "node", "", "the node to exclude this index from (required)")
	cmd.MarkFlagRequired("node")
	return cmd
}

func newIncludeIndexCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	var apply bool
	var node string
	cmd := &cobra.Command{
		Use:   "include-index <index-name> --node <node-name>",
		Short: "Restrict a single index's shards to a node",
		Long:  "Sets index.routing.allocation.include._name on the given index, so Elasticsearch only allocates that index's shards to the given node. Rarely needed; exclude-index is the common case for relieving disk pressure on a specific node.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return setNodeFilter(ctx, opts, "include._name", node, args[0], apply)
		},
	}
	cmd.Flags().BoolVarP(&apply, "yes", "y", false, "apply the change (default is a dry run that only shows what would be sent)")
	cmd.Flags().StringVar(&node, "node", "", "the node to restrict this index to (required)")
	cmd.MarkFlagRequired("node")
	return cmd
}

func newClearNodeFiltersCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	var apply bool
	var index string
	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Remove every exclude/include node filter (reactivate all nodes, or one index with --index)",
		Long:  "Nulls out routing.allocation.exclude/include._name/_ip/_host in both persistent and transient settings, so no node stays cluster-wide excluded or required. Use this to undo exclude-node/include-node/exclude-index/include-index.\n\nWith --index, only clears those settings on the given index instead of the whole cluster.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return clearNodeFilters(ctx, opts, index, apply)
		},
	}
	cmd.Flags().BoolVarP(&apply, "yes", "y", false, "apply the change (default is a dry run that only shows what would be sent)")
	cmd.Flags().StringVar(&index, "index", "", "scope the clear to this index only, instead of the whole cluster")
	return cmd
}
