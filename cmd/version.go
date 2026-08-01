package cmd

import (
	"context"
	"fmt"

	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/spf13/cobra"
)

func newVersionCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version",
		RunE: command.Repeat(ctx, opts, func(cmd *cobra.Command, args []string) error {
			fmt.Println(appVersion)
			return nil
		}),
	}
}
