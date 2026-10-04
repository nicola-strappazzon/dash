package overflow

import (
	"context"
	"fmt"

	"github.com/fatih/color"
	"github.com/nicola-strappazzon/dash/cmd/mysql/internal/runner"
	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/nicola-strappazzon/dash/internal/driver/mysql"
	"github.com/nicola-strappazzon/go-table"
	"github.com/spf13/cobra"
)

func NewCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	return runner.SectionCommand(ctx, opts, "overflow", "Show AUTO_INCREMENT overflow risk", Render(opts))
}

func Render(opts *command.Options) runner.RenderFunc {
	return func(ctx context.Context, db *mysql.MySQL) error {
		if opts.MySQL.Database == "" {
			return fmt.Errorf("missing MySQL database: pass --database")
		}

		columns, err := db.AutoIncrements(ctx, opts.MySQL.Database)
		if err != nil {
			return err
		}

		tbl := table.New()
		tbl.Title("MySQL AUTO_INCREMENT overflow · " + opts.MySQL.Database)
		for _, column := range columns {
			tbl.Add(column.Table, column.Column, column.ColumnType, column.CurrentValue, column.MaxValue, column.UsagePercent, column.Remaining)
		}
		tbl.Column(0, table.Column{Name: "TABLE", MaxWidth: 40})
		tbl.Column(1, table.Column{Name: "COLUMN", MaxWidth: 32})
		tbl.Column(2, table.Column{Name: "TYPE"})
		tbl.Column(3, table.Column{Name: "CURRENT", Alignment: table.Right})
		tbl.Column(4, table.Column{Name: "MAX", Alignment: table.Right})
		tbl.Column(5, table.Column{
			Name:      "USED",
			Format:    table.Percentage,
			Precision: 2,
			Alignment: table.Right,
			Colors: []table.ColorRule{
				{Condition: ">= 90", Color: color.FgRed},
				{Condition: ">= 70", Color: color.FgYellow},
				{Condition: ">= 0", Color: color.FgGreen},
			},
		})
		tbl.Column(6, table.Column{Name: "REMAINING", Alignment: table.Right})
		tbl.Margin(table.Margin{Left: 2})
		tbl.Padding(2)
		tbl.FitWidth(table.TerminalWidth())
		tbl.SetWidth(table.TerminalWidth())
		tbl.Print()
		fmt.Println("")
		return nil
	}
}
