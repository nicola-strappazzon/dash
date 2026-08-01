package shards

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/nicola-strappazzon/dash/cmd/elasticsearch/internal/runner"
	"github.com/nicola-strappazzon/dash/internal/command"
	elasticsearchdriver "github.com/nicola-strappazzon/dash/internal/driver/elasticsearch"
	"github.com/nicola-strappazzon/dash/internal/parse"
	"github.com/nicola-strappazzon/go-table"
	"github.com/spf13/cobra"
)

type shard struct {
	Index  string `json:"index"`
	Shard  string `json:"shard"`
	Prirep string `json:"prirep"`
	State  string `json:"state"`
	Docs   string `json:"docs"`
	Store  string `json:"store"`
	IP     string `json:"ip"`
	Node   string `json:"node"`
}

func NewCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	return runner.SectionCommand(ctx, opts, "shards", "Show shards", Render)
}

func Render(ctx context.Context, es *elasticsearchdriver.Elasticsearch) error {
	res, err := es.Client().Cat.Shards(
		es.Client().Cat.Shards.WithContext(ctx),
		es.Client().Cat.Shards.WithFormat("json"),
		es.Client().Cat.Shards.WithBytes("b"),
		es.Client().Cat.Shards.WithS("state,index,shard"),
		es.Client().Cat.Shards.WithH("index", "shard", "prirep", "state", "docs", "store", "ip", "node"),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error en la respuesta de _cat/shards: %s", res.String())
	}

	var shards []shard
	if err := json.NewDecoder(res.Body).Decode(&shards); err != nil {
		return fmt.Errorf("error al decodificar _cat/shards: %w", err)
	}

	tbl := table.New()
	tbl.Title("Shards")
	for _, s := range shards {
		tbl.Add(
			s.Index,
			parse.Int(s.Shard),
			strings.ToUpper(s.Prirep),
			s.State,
			parse.Int(s.Docs),
			parse.Float(s.Store),
			s.IP,
			s.Node,
		)
	}
	tbl.Column(0, table.Column{Name: "INDEX", Truncate: 34})
	tbl.Column(1, table.Column{Name: "SHARD", Alignment: table.Right, Width: 5})
	tbl.Column(2, table.Column{
		Name:      "P/R",
		Alignment: table.Center,
		Width:     3,
		Colors: []table.ColorRule{
			{Condition: `== "P"`, Color: color.FgGreen},
			{Condition: `== "R"`, Color: color.FgYellow},
		},
	})
	tbl.Column(3, table.Column{
		Name: "STATE",
		Colors: []table.ColorRule{
			{Condition: `== "STARTED"`, Color: color.FgGreen},
			{Condition: `== "INITIALIZING"`, Color: color.FgYellow},
			{Condition: `== "RELOCATING"`, Color: color.FgYellow},
			{Condition: `== "UNASSIGNED"`, Color: color.FgRed},
		},
	})
	tbl.Column(4, table.Column{Name: "DOCS", Alignment: table.Right, Width: 11})
	tbl.Column(5, table.Column{Name: "STORE", Format: table.Bytes, Alignment: table.Right, Width: 10})
	tbl.Column(6, table.Column{Name: "IP"})
	tbl.Column(7, table.Column{Name: "NODE", Truncate: 18})
	tbl.Margin(table.Margin{Left: 2})
	tbl.Padding(3)
	tbl.Print()
	fmt.Println("")

	return nil
}
