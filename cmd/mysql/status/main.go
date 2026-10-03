package status

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
	cmd := runner.SectionCommand(ctx, opts, "status", "List MySQL global status counters", Render(opts))
	cmd.Flags().StringVar(&opts.MySQL.StatusLike, "like", "", "filter status names with a MySQL LIKE pattern, e.g. Innodb%")
	return cmd
}

func Render(opts *command.Options) runner.RenderFunc {
	return func(ctx context.Context, db *mysql.MySQL) error {
		status, err := db.GlobalStatus(ctx, opts.MySQL.StatusLike)
		if err != nil {
			return err
		}

		title := "MySQL global status"
		if opts.MySQL.StatusLike != "" {
			title += " · " + opts.MySQL.StatusLike
		}
		tbl := table.New()
		tbl.Title(title)
		for _, variable := range status {
			tbl.Add(variable.Name, variable.Value)
		}
		tbl.Column(0, table.Column{Name: "STATUS"})
		tbl.Column(1, table.Column{Name: "VALUE", Alignment: table.Right})
		tbl.Margin(table.Margin{Left: 2})
		tbl.Padding(3)
		tbl.SetWidth(table.TerminalWidth())
		tbl.Print()
		fmt.Println("")
		return nil
	}
}
