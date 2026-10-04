package indexes

import (
	"context"
	"fmt"
	"strconv"

	"github.com/fatih/color"
	"github.com/nicola-strappazzon/dash/cmd/mysql/internal/runner"
	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/nicola-strappazzon/dash/internal/driver/mysql"
	"github.com/nicola-strappazzon/go-table"
	"github.com/spf13/cobra"
)

func NewCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	cmd := runner.SectionCommand(ctx, opts, "indexes", "List indexes and details for a MySQL table", Render(opts))
	cmd.Flags().StringVar(&opts.MySQL.Table, "table", "", "MySQL table name")
	return cmd
}

func Render(opts *command.Options) runner.RenderFunc {
	return func(ctx context.Context, db *mysql.MySQL) error {
		if opts.MySQL.Database == "" {
			return fmt.Errorf("missing MySQL database: pass --database")
		}
		if opts.MySQL.Table == "" {
			return fmt.Errorf("missing MySQL table: pass --table")
		}

		indexes, err := db.Indexes(ctx, opts.MySQL.Database, opts.MySQL.Table)
		if err != nil {
			return err
		}

		tbl := table.New()
		tbl.Title("MySQL indexes · " + opts.MySQL.Database + "." + opts.MySQL.Table)
		for _, index := range indexes {
			unique := "NO"
			if index.Unique {
				unique = "YES"
			}
			visible := "NO"
			if index.Visible {
				visible = "YES"
			}
			tbl.Add(index.Name, index.Type, unique, visible, strconv.FormatInt(index.Cardinality, 10), indexSize(index), mysql.FormatIndexColumns(index.Columns))
		}
		tbl.Column(0, table.Column{Name: "INDEX"})
		tbl.Column(1, table.Column{Name: "TYPE"})
		tbl.Column(2, table.Column{Name: "UNIQUE", Color: color.FgGreen})
		tbl.Column(3, table.Column{Name: "VISIBLE", Color: color.FgGreen})
		tbl.Column(4, table.Column{Name: "CARDINALITY", Alignment: table.Right})
		tbl.Column(5, table.Column{Name: "SIZE", Alignment: table.Right})
		tbl.Column(6, table.Column{Name: "COLUMNS", MaxWidth: 70})
		tbl.Margin(table.Margin{Left: 2})
		tbl.Padding(2)
		tbl.FitWidth(table.TerminalWidth())
		tbl.SetWidth(table.TerminalWidth())
		tbl.Print()
		fmt.Println("")
		return nil
	}
}

func indexSize(index mysql.Index) string {
	if !index.SizeKnown {
		return "—"
	}
	return table.Field{Value: index.SizeBytes, Format: table.Bytes}.Render()
}
