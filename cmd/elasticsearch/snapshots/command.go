package snapshotscmd

import (
	"context"

	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/spf13/cobra"
)

func NewCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshots",
		Short: "Show snapshot repositories or repository snapshots",
		Args:  cobra.MaximumNArgs(1),
		RunE: command.Repeat(ctx, opts, func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}

			return renderRepository(ctx, opts, args[0])
		}),
	}

	cmd.AddCommand(
		newRepositoriesCommand(ctx, opts),
		newListCommand(ctx, opts),
	)

	return cmd
}
