package recovery

import (
	"context"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/nicola-strappazzon/dash/cmd/elasticsearch/internal/runner"
	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/nicola-strappazzon/dash/internal/driver/elasticsearch"
	"github.com/nicola-strappazzon/dash/internal/parse"
	"github.com/nicola-strappazzon/go-table"
	"github.com/spf13/cobra"
)

func NewCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	return runner.SectionCommand(ctx, opts, "recovery", "Show active shard recovery", Render)
}

func Render(ctx context.Context, es *elasticsearch.Elasticsearch) error {
	recoveries, err := es.Recovery(ctx)
	if err != nil {
		return err
	}

	tbl := table.New()
	tbl.Title("Recovery")
	for _, r := range recoveries {
		tbl.Add(
			r.Index,
			parse.Int(r.Shard),
			parse.Uptime(r.Time),
			estimatedRecoveryRemaining(r),
			strings.ToUpper(r.Type),
			strings.ToUpper(r.Stage),
			r.SourceNode,
			r.TargetNode,
			parse.Int(r.FilesRecovered),
			parse.Int(r.FilesTotal),
			parse.Percent(r.FilesPercent),
			parse.Float(r.BytesRecovered),
			parse.Float(r.BytesTotal),
			parse.Percent(r.BytesPercent),
			parse.Int(r.TranslogOpsRecovered),
			parse.Int(r.TranslogOps),
			parse.Percent(r.TranslogOpsPercent),
		)
	}
	tbl.Column(0, table.Column{Name: "INDEX", Truncate: 30})
	tbl.Column(1, table.Column{Name: "SHARD", Alignment: table.Right, Width: 5})
	tbl.Column(2, table.Column{Name: "TIME", Format: table.Duration, Alignment: table.Right, Width: 8})
	tbl.Column(3, table.Column{Name: "ETA", Format: table.Duration, Alignment: table.Right, Width: 8})
	tbl.Column(4, table.Column{Name: "TYPE", Truncate: 8})
	tbl.Column(5, table.Column{
		Name: "STAGE",
		Colors: []table.ColorRule{
			{Condition: `== "DONE"`, Color: color.FgGreen},
			{Condition: `== "FINALIZE"`, Color: color.FgYellow},
			{Condition: `== "TRANSLOG"`, Color: color.FgYellow},
			{Condition: `== "INDEX"`, Color: color.FgYellow},
			{Condition: `== "INIT"`, Color: color.FgYellow},
		},
	})
	tbl.Column(6, table.Column{Name: "SOURCE", Truncate: 16})
	tbl.Column(7, table.Column{Name: "TARGET", Truncate: 16})
	tbl.Column(8, table.Column{Name: "FILES", Alignment: table.Right, Width: 7})
	tbl.Column(9, table.Column{Name: "FILES.T", Alignment: table.Right, Width: 7})
	tbl.Column(10, table.Column{Name: "FILES%", Format: table.Percentage, Alignment: table.Right, Width: 7})
	tbl.Column(11, table.Column{Name: "BYTES", Format: table.Bytes, Alignment: table.Right, Width: 10})
	tbl.Column(12, table.Column{Name: "BYTES.T", Format: table.Bytes, Alignment: table.Right, Width: 10})
	tbl.Column(13, table.Column{Name: "BYTES%", Format: table.Percentage, Alignment: table.Right, Width: 7})
	tbl.Column(14, table.Column{Name: "TRANS", Alignment: table.Right, Width: 8})
	tbl.Column(15, table.Column{Name: "TRANS.T", Alignment: table.Right, Width: 8})
	tbl.Column(16, table.Column{Name: "TRANS%", Format: table.Percentage, Alignment: table.Right, Width: 7})
	tbl.Margin(table.Margin{Left: 2})
	tbl.Padding(3)
	tbl.SetWidth(table.TerminalWidth())
	tbl.Print()
	fmt.Println("")

	return nil
}

func estimatedRecoveryRemaining(r elasticsearch.Recovery) float64 {
	elapsed := parse.Uptime(r.Time)
	recovered := parse.Float(r.BytesRecovered)
	total := parse.Float(r.BytesTotal)
	if elapsed <= 0 || recovered <= 0 || total <= recovered {
		return 0
	}

	return elapsed * ((total - recovered) / recovered)
}
