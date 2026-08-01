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

type index struct {
	Health       string `json:"health"`
	Status       string `json:"status"`
	Name         string `json:"index"`
	Primary      string `json:"pri"`
	Replica      string `json:"rep"`
	DocsCount    string `json:"docs.count"`
	DocsDeleted  string `json:"docs.deleted"`
	StoreSize    string `json:"store.size"`
	PriStoreSize string `json:"pri.store.size"`
}

func newIndicesCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	return sectionCommand(ctx, opts, "indices", "Show indices")
}

func renderIndices(ctx context.Context, es *dashboard.Elasticsearch) error {
	res, err := es.Client().Cat.Indices(
		es.Client().Cat.Indices.WithContext(ctx),
		es.Client().Cat.Indices.WithFormat("json"),
		es.Client().Cat.Indices.WithBytes("b"),
		es.Client().Cat.Indices.WithS("store.size:desc"),
		es.Client().Cat.Indices.WithH("health", "status", "index", "pri", "rep", "docs.count", "docs.deleted", "store.size", "pri.store.size"),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error en la respuesta de _cat/indices: %s", res.String())
	}

	var indices []index
	if err := json.NewDecoder(res.Body).Decode(&indices); err != nil {
		return fmt.Errorf("error al decodificar _cat/indices: %w", err)
	}

	tbl := table.New()
	tbl.Title("Indices")
	for _, i := range indices {
		tbl.Add(
			i.Name,
			strings.ToUpper(i.Health),
			i.Status,
			common.ParseInt(i.Primary),
			common.ParseInt(i.Replica),
			common.ParseInt(i.DocsCount),
			common.ParseInt(i.DocsDeleted),
			common.ParseFloat(i.StoreSize),
			common.ParseFloat(i.PriStoreSize),
		)
	}
	tbl.Column(0, table.Column{Name: "NAME", Truncate: 38})
	tbl.Column(1, table.Column{
		Name: "HEALTH",
		Colors: []table.ColorRule{
			{Condition: `== "GREEN"`, Color: color.FgGreen},
			{Condition: `== "YELLOW"`, Color: color.FgYellow},
			{Condition: `== "RED"`, Color: color.FgRed},
		},
	})
	tbl.Column(2, table.Column{
		Name: "STATUS",
		Colors: []table.ColorRule{
			{Condition: `== "open"`, Color: color.FgGreen},
			{Condition: `== "close"`, Color: color.FgYellow},
		},
	})
	tbl.Column(3, table.Column{Name: "PRI", Alignment: table.Right, Width: 3})
	tbl.Column(4, table.Column{Name: "REP", Alignment: table.Right, Width: 3})
	tbl.Column(5, table.Column{
		Name:      "DOCS",
		Alignment: table.Right,
		Width:     12,
		ZeroFill:  true,
		Precision: 12,
		Scale:     0,
	})
	tbl.Column(6, table.Column{
		Name:      "DELETED",
		Alignment: table.Right,
		Width:     11,
		Color:     color.FgGreen,
		ZeroFill:  true,
		Precision: 11,
		Scale:     0,
		Colors: []table.ColorRule{
			{Condition: "> 0", Color: color.FgYellow},
		},
	})
	tbl.Column(7, table.Column{Name: "STORE", Format: table.Bytes, Alignment: table.Right, Width: 10})
	tbl.Column(8, table.Column{Name: "PRI.STORE", Format: table.Bytes, Alignment: table.Right, Width: 10})
	tbl.Total(5)
	tbl.Total(6)
	tbl.Total(7)
	tbl.Total(8)
	tbl.Margin(table.Margin{Left: 2})
	tbl.Padding(3)
	tbl.Print()
	fmt.Println("")

	return nil
}
