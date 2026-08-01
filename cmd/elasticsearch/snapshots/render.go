package snapshots

import (
	"context"
	"fmt"

	"github.com/fatih/color"
	"github.com/nicola-strappazzon/dash/cmd/elasticsearch/internal/runner"
	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/nicola-strappazzon/dash/internal/driver/elasticsearch"
	"github.com/nicola-strappazzon/dash/internal/parse"
	"github.com/nicola-strappazzon/go-table"
)

type snapshotRepository struct {
	Name     string
	Type     string
	Location string
	BasePath string
}

func renderRepositories(ctx context.Context, opts *command.Options) error {
	return runner.WithElasticsearch(opts, func(es *elasticsearch.Elasticsearch) error {
		repositoryResponse, err := es.SnapshotRepositories(ctx)
		if err != nil {
			return err
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
		tbl.Padding(3)
		tbl.SortBy(0)
		tbl.SetWidth(table.TerminalWidth())
		tbl.Print()
		fmt.Println("")

		return nil
	})
}

func renderRepository(ctx context.Context, opts *command.Options, repository string) error {
	return runner.WithElasticsearch(opts, func(es *elasticsearch.Elasticsearch) error {
		return Render(ctx, es, repository)
	})
}

func Repository(repository string) runner.RenderFunc {
	return func(ctx context.Context, es *elasticsearch.Elasticsearch) error {
		return Render(ctx, es, repository)
	}
}

func Render(ctx context.Context, es *elasticsearch.Elasticsearch, repository string) error {
	snapshots, err := es.Snapshots(ctx, repository)
	if err != nil {
		return err
	}

	tbl := table.New()
	tbl.Title("Snapshots")
	for _, s := range snapshots {
		tbl.Add(
			s.ID,
			s.Status,
			s.StartTime,
			s.Duration,
			parse.Float(s.Indices),
			parse.Float(s.SuccessfulShards),
			parse.Float(s.FailedShards),
			parse.Float(s.TotalShards),
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
	tbl.Padding(3)
	tbl.SetWidth(table.TerminalWidth())
	tbl.Print()
	fmt.Println("")

	return nil
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
