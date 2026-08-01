package snapshotscmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fatih/color"
	"github.com/nicola-strappazzon/go-table"
	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/nicola-strappazzon/dash/internal/common"
	"github.com/nicola-strappazzon/dash/internal/dashboard"
)

type snapshotRepository struct {
	Name     string
	Type     string
	Location string
	BasePath string
}

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

func renderRepositories(ctx context.Context, opts *command.Options) error {
	return withElasticsearch(opts, func(es *dashboard.Elasticsearch) error {
		res, err := es.Client().Snapshot.GetRepository(
			es.Client().Snapshot.GetRepository.WithContext(ctx),
		)
		if err != nil {
			return err
		}
		defer res.Body.Close()

		if res.IsError() {
			return fmt.Errorf("error en la respuesta de _snapshot: %s", res.String())
		}

		var repositoryResponse map[string]struct {
			Type     string         `json:"type"`
			Settings map[string]any `json:"settings"`
		}
		if err := json.NewDecoder(res.Body).Decode(&repositoryResponse); err != nil {
			return fmt.Errorf("error al decodificar _snapshot: %w", err)
		}

		repositories := make([]snapshotRepository, 0, len(repositoryResponse))
		for name, repository := range repositoryResponse {
			repositories = append(repositories, snapshotRepository{
				Name:     name,
				Type:     repository.Type,
				Location: formatStringSetting(repository.Settings, "location"),
				BasePath: formatStringSetting(repository.Settings, "base_path"),
			})
		}

		tbl := table.New()
		tbl.Title("Snapshot repositories")
		for _, repository := range repositories {
			tbl.Add(
				repository.Name,
				repository.Type,
				repository.Location,
				repository.BasePath,
			)
		}
		tbl.Column(0, table.Column{Name: "NAME"})
		tbl.Column(1, table.Column{Name: "TYPE"})
		tbl.Column(2, table.Column{Name: "LOCATION"})
		tbl.Column(3, table.Column{Name: "BASE_PATH"})
		tbl.Margin(table.Margin{Left: 2})
		tbl.SortBy(0)
		tbl.Print()
		fmt.Println("")

		return nil
	})
}

func renderRepository(ctx context.Context, opts *command.Options, repository string) error {
	return withElasticsearch(opts, func(es *dashboard.Elasticsearch) error {
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
	})
}

func formatStringSetting(settings map[string]any, key string) string {
	if value, ok := settings[key]; ok {
		switch v := value.(type) {
		case string:
			if v != "" {
				return v
			}
		case fmt.Stringer:
			return v.String()
		}
	}

	return "-"
}
