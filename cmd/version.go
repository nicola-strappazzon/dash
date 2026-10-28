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
			if command.JSON(ctx) {
				return command.PrintJSON(map[string]string{"version": appVersion})
			}
			fmt.Println(appVersion)
			return nil
		}),
	}
}
