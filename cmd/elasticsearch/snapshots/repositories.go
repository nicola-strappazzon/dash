package snapshots

import (
	"context"

	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/spf13/cobra"
)

func newRepositoriesCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	return &cobra.Command{
		Use:   "repositories",
		Short: "Show snapshot repositories",
		Args:  cobra.NoArgs,
		RunE: command.Repeat(ctx, opts, func(cmd *cobra.Command, args []string) error {
			return renderRepositories(ctx, opts)
		}),
	}
}
