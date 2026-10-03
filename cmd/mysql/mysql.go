package mysql

import (
	"context"

	"github.com/nicola-strappazzon/dash/cmd/mysql/health"
	"github.com/nicola-strappazzon/dash/cmd/mysql/innodb"
	"github.com/nicola-strappazzon/dash/cmd/mysql/internal/runner"
	"github.com/nicola-strappazzon/dash/cmd/mysql/variables"
	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/spf13/cobra"
)

func NewCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mysql [health|innodb|variables...]",
		Short: "MySQL dashboard",
		Args:  cobra.ArbitraryArgs,
		RunE: command.Repeat(ctx, opts, func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return runner.RenderSections(ctx, opts, args)
		}),
	}
	// Reserve -h for the host, matching common MySQL client usage. Cobra's
	// default help shorthand is therefore moved to -?, while --help remains.
	cmd.PersistentFlags().BoolP("help", "?", false, "help for mysql")
	cmd.PersistentFlags().StringVarP(&opts.MySQL.Host, "host", "h", "127.0.0.1:3306", "MySQL server host and port")
	cmd.PersistentFlags().StringVarP(&opts.MySQL.Username, "username", "u", "", "MySQL username")
	cmd.PersistentFlags().StringVarP(&opts.MySQL.Password, "password", "p", "", "MySQL password")
	cmd.PersistentFlags().StringVarP(&opts.MySQL.Database, "database", "d", "", "MySQL database")
	cmd.PersistentFlags().BoolVar(&opts.MySQL.TLS, "tls", false, "enable TLS for the MySQL server")
	cmd.PersistentFlags().BoolVar(&opts.MySQL.InsecureTLS, "insecure-skip-verify", false, "skip MySQL TLS certificate verification")

	cmd.AddCommand(
		health.NewCommand(ctx, opts),
		innodb.NewCommand(ctx, opts),
		variables.NewCommand(ctx, opts),
	)
	return cmd
}
