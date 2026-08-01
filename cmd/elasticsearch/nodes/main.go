package nodes

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fatih/color"
	"github.com/nicola-strappazzon/dash/cmd/elasticsearch/internal/runner"
	"github.com/nicola-strappazzon/dash/internal/command"
	elasticsearchdriver "github.com/nicola-strappazzon/dash/internal/driver/elasticsearch"
	"github.com/nicola-strappazzon/dash/internal/parse"
	"github.com/nicola-strappazzon/go-table"
	"github.com/spf13/cobra"
)

type node struct {
	ID              string `json:"id"`
	IP              string `json:"ip"`
	Name            string `json:"name"`
	Master          string `json:"master"`
	NodeRole        string `json:"node.role"`
	HeapPercent     string `json:"heap.percent"`
	RAMPercent      string `json:"ram.percent"`
	CPU             string `json:"cpu"`
	Load1m          string `json:"load_1m"`
	DiskUsedPercent string `json:"disk.used_percent"`
	Uptime          string `json:"uptime"`
}

func NewCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	return runner.SectionCommand(ctx, opts, "nodes", "Show cluster nodes", Render)
}

func Render(ctx context.Context, es *elasticsearchdriver.Elasticsearch) error {
	res, err := es.Client().Cat.Nodes(
		es.Client().Cat.Nodes.WithContext(ctx),
		es.Client().Cat.Nodes.WithFormat("json"),
		es.Client().Cat.Nodes.WithH("id", "ip", "name", "master", "node.role", "heap.percent", "ram.percent", "cpu", "load_1m", "disk.used_percent", "uptime"),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error en la respuesta de _cat/nodes: %s", res.String())
	}

	var nodes []node
	if err := json.NewDecoder(res.Body).Decode(&nodes); err != nil {
		return fmt.Errorf("error al decodificar _cat/nodes: %w", err)
	}

	tbl := table.New()
	tbl.Title("Cluster nodes")
	for _, n := range nodes {
		tbl.Add(
			n.ID,
			n.Name,
			n.IP,
			n.Master,
			parse.Float(n.HeapPercent),
			parse.Float(n.RAMPercent),
			parse.Float(n.CPU),
			parse.Float(n.DiskUsedPercent),
			parse.Uptime(n.Uptime),
		)
	}
	tbl.Column(0, table.Column{Name: "ID"})
	tbl.Column(1, table.Column{Name: "NAME"})
	tbl.Column(2, table.Column{Name: "IP"})
	tbl.Column(3, table.Column{Name: "MASTER", Alignment: table.Center})
	tbl.Column(4, table.Column{Name: "HEAP", Format: table.Percentage, Alignment: table.Right, Width: 5})
	tbl.Column(5, table.Column{Name: "RAM", Format: table.Percentage, Alignment: table.Right, Width: 5})
	tbl.Column(6, table.Column{
		Name:      "CPU",
		Format:    table.Percentage,
		Alignment: table.Right,
		Width:     6,
		Color:     color.FgGreen,
		Colors: []table.ColorRule{
			{Condition: ">= 80", Color: color.FgRed},
			{Condition: ">= 70", Color: color.FgYellow},
		},
	})
	tbl.Column(7, table.Column{
		Name:      "DISK",
		Format:    table.Percentage,
		Alignment: table.Right,
		Width:     8,
		Color:     color.FgGreen,
		Colors: []table.ColorRule{
			{Condition: ">= 90", Color: color.FgRed},
			{Condition: ">= 80", Color: color.FgYellow},
		},
	})
	tbl.Column(8, table.Column{
		Name:      "UPTIME",
		Format:    table.Duration,
		Width:     9,
		Alignment: table.Right,
		Color:     color.FgGreen,
		Colors: []table.ColorRule{
			{Condition: "< 3600", Color: color.FgRed},
			{Condition: "< 86400", Color: color.FgYellow},
		},
	})
	tbl.Margin(table.Margin{Left: 2})
	tbl.SortBy(1)
	tbl.Print()
	fmt.Println("")

	return nil
}
