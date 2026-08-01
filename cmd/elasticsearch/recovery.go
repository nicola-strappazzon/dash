package elasticsearchcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/nicola-strappazzon/go-table"
	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/nicola-strappazzon/dash/internal/common"
	"github.com/nicola-strappazzon/dash/internal/dashboard"
	"github.com/spf13/cobra"
)

type recovery struct {
	Index                string `json:"index"`
	Shard                string `json:"shard"`
	Time                 string `json:"time"`
	Type                 string `json:"type"`
	Stage                string `json:"stage"`
	SourceNode           string `json:"source_node"`
	TargetNode           string `json:"target_node"`
	FilesRecovered       string `json:"files_recovered"`
	FilesTotal           string `json:"files_total"`
	FilesPercent         string `json:"files_percent"`
	BytesRecovered       string `json:"bytes_recovered"`
	BytesTotal           string `json:"bytes_total"`
	BytesPercent         string `json:"bytes_percent"`
	TranslogOpsRecovered string `json:"translog_ops_recovered"`
	TranslogOps          string `json:"translog_ops"`
	TranslogOpsPercent   string `json:"translog_ops_percent"`
}

func newRecoveryCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	return sectionCommand(ctx, opts, "recovery", "Show active shard recovery")
}

func renderRecovery(ctx context.Context, es *dashboard.Elasticsearch) error {
	res, err := es.Client().Cat.Recovery(
		es.Client().Cat.Recovery.WithContext(ctx),
		es.Client().Cat.Recovery.WithFormat("json"),
		es.Client().Cat.Recovery.WithActiveOnly(true),
		es.Client().Cat.Recovery.WithBytes("b"),
		es.Client().Cat.Recovery.WithTime("s"),
		es.Client().Cat.Recovery.WithS("index,shard"),
		es.Client().Cat.Recovery.WithH(
			"index",
			"shard",
			"time",
			"type",
			"stage",
			"source_node",
			"target_node",
			"files_recovered",
			"files_total",
			"files_percent",
			"bytes_recovered",
			"bytes_total",
			"bytes_percent",
			"translog_ops_recovered",
			"translog_ops",
			"translog_ops_percent",
		),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error en la respuesta de _cat/recovery: %s", res.String())
	}

	var recoveries []recovery
	if err := json.NewDecoder(res.Body).Decode(&recoveries); err != nil {
		return fmt.Errorf("error al decodificar _cat/recovery: %w", err)
	}

	tbl := table.New()
	tbl.Title("Recovery")
	for _, r := range recoveries {
		tbl.Add(
			r.Index,
			common.ParseInt(r.Shard),
			common.ParseUptime(r.Time),
			estimatedRecoveryRemaining(r),
			strings.ToUpper(r.Type),
			strings.ToUpper(r.Stage),
			r.SourceNode,
			r.TargetNode,
			common.ParseInt(r.FilesRecovered),
			common.ParseInt(r.FilesTotal),
			parsePercent(r.FilesPercent),
			common.ParseFloat(r.BytesRecovered),
			common.ParseFloat(r.BytesTotal),
			parsePercent(r.BytesPercent),
			common.ParseInt(r.TranslogOpsRecovered),
			common.ParseInt(r.TranslogOps),
			parsePercent(r.TranslogOpsPercent),
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
	tbl.Print()
	fmt.Println("")

	return nil
}

func parsePercent(s string) float64 {
	return common.ParseFloat(strings.TrimSuffix(strings.TrimSpace(s), "%"))
}

func estimatedRecoveryRemaining(r recovery) float64 {
	elapsed := common.ParseUptime(r.Time)
	recovered := common.ParseFloat(r.BytesRecovered)
	total := common.ParseFloat(r.BytesTotal)
	if elapsed <= 0 || recovered <= 0 || total <= recovered {
		return 0
	}

	return elapsed * ((total - recovered) / recovered)
}
