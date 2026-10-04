package tables

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
	return runner.SectionCommand(ctx, opts, "tables", "List tables in a MySQL database", Render(opts))
}

func Render(opts *command.Options) runner.RenderFunc {
	return func(ctx context.Context, db *mysql.MySQL) error {
		if opts.MySQL.Database == "" {
			return fmt.Errorf("missing MySQL database: pass --database")
		}

		tables, err := db.Tables(ctx, opts.MySQL.Database)
		if err != nil {
			return err
		}

		tbl := table.New()
		tbl.Title("MySQL tables · " + opts.MySQL.Database)
		for _, tableInfo := range tables {
			tbl.Add(
				tableInfo.Name,
				tableInfo.Engine,
				tableInfo.Rows,
				tableInfo.DataBytes,
				tableInfo.IndexBytes,
				tableInfo.DataBytes+tableInfo.IndexBytes,
				tableInfo.FreeBytes,
				tableInfo.FragmentationPercent,
			)
		}
		tbl.Column(0, table.Column{Name: "TABLE", MaxWidth: 60})
		tbl.Column(1, table.Column{Name: "ENGINE"})
		tbl.Column(2, table.Column{Name: "ROWS", Alignment: table.Right, Width: 8})
		tbl.Column(3, table.Column{Name: "DATA", Format: table.Bytes, Alignment: table.Right, Width: 9})
		tbl.Column(4, table.Column{Name: "INDEX", Format: table.Bytes, Alignment: table.Right, Width: 9})
		tbl.Column(5, table.Column{Name: "TOTAL", Format: table.Bytes, Alignment: table.Right, Width: 9})
		tbl.Column(6, table.Column{Name: "FREE", Format: table.Bytes, Alignment: table.Right, Width: 9})
		tbl.Column(7, table.Column{
			Name:      "FRAG%",
			Format:    table.Percentage,
			Alignment: table.Right,
			Width:     6,
			Color:     color.FgGreen,
			Colors: []table.ColorRule{
				{Condition: ">= 30", Color: color.FgRed},
				{Condition: ">= 10", Color: color.FgYellow},
			},
		})
		tbl.Margin(table.Margin{Left: 2})
		tbl.Padding(2)
		tbl.FitWidth(table.TerminalWidth())
		tbl.SetWidth(table.TerminalWidth())
		tbl.Print()
		fmt.Println("")
		return nil
	}
}
