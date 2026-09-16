package clickhouse

import (
	"context"

	"github.com/nicola-strappazzon/dash/cmd/clickhouse/disk"
	"github.com/nicola-strappazzon/dash/cmd/clickhouse/internal/runner"
	"github.com/nicola-strappazzon/dash/cmd/clickhouse/replica"
	"github.com/nicola-strappazzon/dash/cmd/clickhouse/uptime"
	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/spf13/cobra"
)

func NewCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clickhouse [uptime|replica|disk...]",
		Short: "ClickHouse dashboard",
		Args:  cobra.ArbitraryArgs,
		RunE: command.Repeat(ctx, opts, func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}

			return runner.RenderSections(ctx, opts, args)
		}),
	}
	cmd.PersistentFlags().StringVar(&opts.ClickHouse.Host, "host", "127.0.0.1:9000", "ClickHouse server host and port")
	cmd.PersistentFlags().StringVarP(&opts.ClickHouse.Username, "username", "u", "default", "ClickHouse username")
	cmd.PersistentFlags().StringVarP(&opts.ClickHouse.Password, "password", "p", "", "ClickHouse password")
	cmd.PersistentFlags().BoolVar(&opts.ClickHouse.TLS, "tls", false, "enable TLS for the ClickHouse server")
	cmd.PersistentFlags().BoolVar(&opts.ClickHouse.InsecureTLS, "insecure-skip-verify", false, "skip ClickHouse TLS certificate verification")
	cmd.PersistentFlags().StringVar(&opts.ClickHouse.ClusterName, "cluster-name", "", "ClickHouse cluster name, e.g. as seen in system.clusters")

	cmd.AddCommand(
		disk.NewCommand(ctx, opts),
		replica.NewCommand(ctx, opts),
		uptime.NewCommand(ctx, opts),
	)

	return cmd
}
