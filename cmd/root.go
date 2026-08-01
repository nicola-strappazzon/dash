package cmd

import (
	"context"

	elasticsearchcmd "github.com/nicola-strappazzon/dash/cmd/elasticsearch"
	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/spf13/cobra"
)

const appVersion = "0.1.0"

func NewRootCommand(ctx context.Context) *cobra.Command {
	opts := &command.Options{}
	cmd := &cobra.Command{
		Use:   "dash",
		Short: "Terminal services dashboard",
	}
	cmd.PersistentFlags().DurationVarP(&opts.Watch, "watch", "w", 0, "repeat command every duration, e.g. 5s or 1m")
	cmd.PersistentFlags().BoolVarP(&opts.Clear, "clear", "c", false, "clear terminal before each run")

	cmd.AddCommand(
		elasticsearchcmd.NewCommand(ctx, opts),
		newVersionCommand(ctx, opts),
	)

	return cmd
}
