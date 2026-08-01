package snapshots

import (
	"context"

	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/spf13/cobra"
)

func newListCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	return &cobra.Command{
		Use:   "list <repository>",
		Short: "Show snapshots in a repository",
		Args:  cobra.ExactArgs(1),
		RunE: command.Repeat(ctx, opts, func(cmd *cobra.Command, args []string) error {
			return renderRepository(ctx, opts, args[0])
		}),
	}
}
