package databases

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
	return runner.SectionCommand(ctx, opts, "databases", "List MySQL databases", Render)
}

func Render(ctx context.Context, db *mysql.MySQL) error {
	databases, err := db.Databases(ctx)
	if err != nil {
		return err
	}

	tbl := table.New()
	tbl.Title("MySQL databases")
	for _, database := range databases {
		tbl.Add(database.Name, database.Tables, database.SizeBytes, database.Charset, database.Collation)
	}
	tbl.Column(0, table.Column{Name: "DATABASE"})
	tbl.Column(1, table.Column{Name: "TABLES", Alignment: table.Right, Width: 6})
	tbl.Column(2, table.Column{Name: "SIZE", Format: table.Bytes, Alignment: table.Right, Width: 10})
	tbl.Column(3, table.Column{Name: "CHARSET"})
	tbl.Column(4, table.Column{Name: "COLLATION"})
	tbl.Margin(table.Margin{Left: 2})
	tbl.Padding(3)
	tbl.SetWidth(table.TerminalWidth())
	tbl.Print()
	fmt.Println("")
	return nil
}
