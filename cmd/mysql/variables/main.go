package variables

import (
	"context"

	"github.com/nicola-strappazzon/dash/cmd/mysql/internal/runner"
	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/nicola-strappazzon/dash/internal/driver/mysql"
	"github.com/nicola-strappazzon/go-table"
	"github.com/spf13/cobra"
)

func NewCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	cmd := runner.SectionCommand(ctx, opts, "variables", "List MySQL global variables", Render(opts))
	cmd.Flags().StringVar(&opts.MySQL.VariableLike, "like", "", "filter variable names with a MySQL LIKE pattern, e.g. innodb%")
	return cmd
}

func Render(opts *command.Options) runner.RenderFunc {
	return func(ctx context.Context, db *mysql.MySQL) error {
		variables, err := db.Variables(ctx, opts.MySQL.VariableLike)
		if err != nil {
			return err
		}

		title := "MySQL global variables"
		if opts.MySQL.VariableLike != "" {
			title += " · " + opts.MySQL.VariableLike
		}
		tbl := command.NewTable()
		tbl.Title(title)
		for _, variable := range variables {
			tbl.Add(variable.Name, variable.Value)
		}
		tbl.Column(0, table.Column{Name: "VARIABLE"})
		tbl.Column(1, table.Column{Name: "VALUE", MaxWidth: table.TerminalWidth()})
		tbl.Margin(table.Margin{Left: 2})
		tbl.Padding(3)
		tbl.FitWidth(table.TerminalWidth())
		tbl.SetWidth(table.TerminalWidth())
		return command.Print(ctx, tbl)
	}
}
