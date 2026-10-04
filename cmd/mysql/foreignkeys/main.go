package foreignkeys

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
	cmd := runner.SectionCommand(ctx, opts, "foreign-keys", "List foreign keys for a MySQL table", Render(opts))
	cmd.Flags().StringVar(&opts.MySQL.Table, "table", "", "MySQL table name (lists all foreign keys when omitted)")
	return cmd
}

func Render(opts *command.Options) runner.RenderFunc {
	return func(ctx context.Context, db *mysql.MySQL) error {
		if opts.MySQL.Database == "" {
			return fmt.Errorf("missing MySQL database: pass --database")
		}
		if opts.MySQL.Table == "" {
			keys, err := db.AllForeignKeys(ctx, opts.MySQL.Database)
			if err != nil {
				return err
			}
			printAllTable("MySQL foreign keys · "+opts.MySQL.Database, keys)
			return nil
		}

		report, err := db.ForeignKeys(ctx, opts.MySQL.Database, opts.MySQL.Table)
		if err != nil {
			return err
		}
		printTable("Foreign keys · "+opts.MySQL.Database+"."+opts.MySQL.Table+" → references", report.Outgoing)
		printTable("Foreign keys · "+opts.MySQL.Database+"."+opts.MySQL.Table+" ← referenced by", report.Incoming)
		return nil
	}
}

func printAllTable(title string, keys []mysql.ForeignKey) {
	tbl := table.New()
	tbl.Title(title)
	for _, key := range keys {
		tbl.Add(key.Constraint, key.Table, key.Column, key.OtherTable, key.OtherColumn, key.OnUpdate, key.OnDelete)
	}
	tbl.Column(0, table.Column{Name: "CONSTRAINT", MaxWidth: 50})
	tbl.Column(1, table.Column{Name: "TABLE", MaxWidth: 40})
	tbl.Column(2, table.Column{Name: "COLUMN", MaxWidth: 32})
	tbl.Column(3, table.Column{Name: "REFERENCED TABLE", MaxWidth: 40})
	tbl.Column(4, table.Column{Name: "REFERENCED COLUMN", MaxWidth: 32})
	tbl.Column(5, table.Column{Name: "ON UPDATE"})
	tbl.Column(6, table.Column{Name: "ON DELETE"})
	tbl.Margin(table.Margin{Left: 2})
	tbl.Padding(2)
	tbl.FitWidth(table.TerminalWidth())
	tbl.SetWidth(table.TerminalWidth())
	tbl.Print()
	fmt.Println("")
}

func printTable(title string, keys []mysql.ForeignKey) {
	tbl := table.New()
	tbl.Title(title)
	for _, key := range keys {
		tbl.Add(key.Constraint, key.Column, key.OtherTable, key.OtherColumn, key.OnUpdate, key.OnDelete)
	}
	tbl.Column(0, table.Column{Name: "CONSTRAINT", MaxWidth: 50})
	tbl.Column(1, table.Column{Name: "COLUMN", MaxWidth: 32})
	tbl.Column(2, table.Column{Name: "OTHER TABLE", MaxWidth: 40})
	tbl.Column(3, table.Column{Name: "OTHER COLUMN", MaxWidth: 32})
	tbl.Column(4, table.Column{Name: "ON UPDATE"})
	tbl.Column(5, table.Column{Name: "ON DELETE"})
	tbl.Margin(table.Margin{Left: 2})
	tbl.Padding(2)
	tbl.FitWidth(table.TerminalWidth())
	tbl.SetWidth(table.TerminalWidth())
	tbl.Print()
	fmt.Println("")
}
