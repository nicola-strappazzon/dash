package cmd

import (
	"context"

	"github.com/nicola-strappazzon/dash/cmd/clickhouse"
	"github.com/nicola-strappazzon/dash/cmd/elasticsearch"
	"github.com/nicola-strappazzon/dash/cmd/mysql"
	"github.com/nicola-strappazzon/dash/internal/command"

	"github.com/spf13/cobra"
)

const appVersion = "0.1.0"

func NewRootCommand(ctx context.Context) *cobra.Command {
	opts := &command.Options{}
	ctx = command.WithOutput(ctx, opts)
	cmd := &cobra.Command{
		Use:           "dash",
		Short:         "Terminal services dashboard",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.PersistentFlags().DurationVarP(&opts.Watch, "watch", "w", 0, "repeat command every duration, e.g. 5s or 1m")
	cmd.PersistentFlags().BoolVarP(&opts.Clear, "clear", "c", false, "clear terminal before each run")
	cmd.PersistentFlags().StringVar(&opts.Format, "format", "table", "output format: table, json or csv")
	cmd.PersistentFlags().StringVar(&opts.Metric, "metric", "", "select comma-separated metric names or columns (case-insensitive; raw values)")
	cmd.PersistentFlags().BoolVar(&opts.Timestamp, "timestamp", false, "prepend a Unix timestamp in seconds to CSV records")

	cmd.PersistentFlags().BoolVar(&opts.Delta, "delta", false, "show changes between watch samples for known counters; keep gauges unchanged")

	cmd.RegisterFlagCompletionFunc("format", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return []string{"table", "json", "csv"}, cobra.ShellCompDirectiveNoFileComp
	})

	cmd.AddCommand(
		clickhouse.NewCommand(ctx, opts),
		elasticsearch.NewCommand(ctx, opts),
		mysql.NewCommand(ctx, opts),
		newVersionCommand(ctx, opts),
	)

	return cmd
}
