package processlist

import (
	"context"
	"fmt"

	"github.com/nicola-strappazzon/dash/cmd/mysql/internal/runner"
	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/nicola-strappazzon/dash/internal/driver/mysql"
	"github.com/nicola-strappazzon/go-table"
	"github.com/spf13/cobra"
)

func NewCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	cmd := runner.SectionCommand(ctx, opts, "processlist", "List MySQL server processes", Render(opts))
	cmd.Flags().BoolVar(&opts.MySQL.IncludeIdle, "idle", false, "include idle Sleep connections")
	cmd.Flags().Int64Var(&opts.MySQL.MinProcessTime, "seconds", 0, "only include processes with TIME >= this many seconds")
	return cmd
}

func Render(opts *command.Options) runner.RenderFunc {
	return func(ctx context.Context, db *mysql.MySQL) error {
		processes, err := db.Processlist(ctx, opts.MySQL.IncludeIdle, opts.MySQL.MinProcessTime)
		if err != nil {
			return err
		}

		tbl := table.New()
		tbl.Title("MySQL processlist")
		for _, process := range processes {
			tbl.Add(process.ID, process.User, process.Host, process.Database, process.Command, process.Time, process.State, process.Info)
		}
		tbl.Column(0, table.Column{Name: "ID", Alignment: table.Right})
		tbl.Column(1, table.Column{Name: "USER"})
		tbl.Column(2, table.Column{Name: "HOST", MaxWidth: 32})
		tbl.Column(3, table.Column{Name: "DATABASE", MaxWidth: 24})
		tbl.Column(4, table.Column{Name: "COMMAND"})
		tbl.Column(5, table.Column{Name: "TIME", Format: table.Duration, Alignment: table.Right})
		tbl.Column(6, table.Column{Name: "STATE", MaxWidth: 32})
		tbl.Column(7, table.Column{Name: "INFO", MaxWidth: 80})
		tbl.Margin(table.Margin{Left: 2})
		tbl.Padding(2)
		tbl.FitWidth(table.TerminalWidth())
		tbl.SetWidth(table.TerminalWidth())
		tbl.Print()
		fmt.Println("")
		return nil
	}
}
