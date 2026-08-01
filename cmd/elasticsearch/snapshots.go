package elasticsearchcmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fatih/color"
	"github.com/nicola-strappazzon/go-table"
	"github.com/nicola-strappazzon/dash/internal/common"
	"github.com/nicola-strappazzon/dash/internal/dashboard"
)

type snapshot struct {
	ID               string `json:"id"`
	Status           string `json:"status"`
	StartEpoch       string `json:"start_epoch"`
	StartTime        string `json:"start_time"`
	EndEpoch         string `json:"end_epoch"`
	EndTime          string `json:"end_time"`
	Duration         string `json:"duration"`
	Indices          string `json:"indices"`
	SuccessfulShards string `json:"successful_shards"`
	FailedShards     string `json:"failed_shards"`
	TotalShards      string `json:"total_shards"`
}

func renderSnapshots(ctx context.Context, es *dashboard.Elasticsearch, repository string) error {
	res, err := es.Client().Cat.Snapshots(
		es.Client().Cat.Snapshots.WithContext(ctx),
		es.Client().Cat.Snapshots.WithRepository(repository),
		es.Client().Cat.Snapshots.WithFormat("json"),
		es.Client().Cat.Snapshots.WithS("start_epoch"),
		es.Client().Cat.Snapshots.WithH("id", "status", "start_time", "end_time", "duration", "indices", "successful_shards", "failed_shards", "total_shards"),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error en la respuesta de _cat/snapshots: %s", res.String())
	}

	var snapshots []snapshot
	if err := json.NewDecoder(res.Body).Decode(&snapshots); err != nil {
		return fmt.Errorf("error al decodificar _cat/snapshots: %w", err)
	}

	tbl := table.New()
	tbl.Title("Snapshots")
	for _, s := range snapshots {
		tbl.Add(
			s.ID,
			s.Status,
			s.StartTime,
			s.Duration,
			common.ParseFloat(s.Indices),
			common.ParseFloat(s.SuccessfulShards),
			common.ParseFloat(s.FailedShards),
			common.ParseFloat(s.TotalShards),
		)
	}
	tbl.Column(0, table.Column{Name: "NAME"})
	tbl.Column(1, table.Column{
		Name: "STATUS",
		Colors: []table.ColorRule{
			{Condition: `== "SUCCESS"`, Color: color.FgGreen},
			{Condition: `== "IN_PROGRESS"`, Color: color.FgYellow},
			{Condition: `== "PARTIAL"`, Color: color.FgYellow},
			{Condition: `== "FAILED"`, Color: color.FgRed},
		},
	})
	tbl.Column(2, table.Column{Name: "START"})
	tbl.Column(3, table.Column{Name: "DUR", Alignment: table.Right, Width: 8})
	tbl.Column(4, table.Column{Name: "IDX", Alignment: table.Right, Width: 5})
	tbl.Column(5, table.Column{Name: "OK", Alignment: table.Right, Width: 5})
	tbl.Column(6, table.Column{
		Name:      "FAIL",
		Alignment: table.Right,
		Color:     color.FgGreen,
		Colors: []table.ColorRule{
			{Condition: "> 0", Color: color.FgRed},
		},
		Width: 5,
	})
	tbl.Column(7, table.Column{Name: "SHARDS", Alignment: table.Right, Width: 7})
	tbl.Margin(table.Margin{Left: 2})
	tbl.Print()
	fmt.Println("")

	return nil
}
