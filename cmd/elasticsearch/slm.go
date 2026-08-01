package elasticsearchcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/nicola-strappazzon/go-table"
	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/nicola-strappazzon/dash/internal/dashboard"
	"github.com/spf13/cobra"
)

type slmPolicy struct {
	Name             string
	Schedule         string
	Repository       string
	Retention        string
	NextExecution    string
	LastSuccess      string
	LastFailure      string
	SnapshotsTaken   int64
	SnapshotsFailed  int64
	SnapshotsDeleted int64
	DeletionFailures int64
}

func newSLMCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	return sectionCommand(ctx, opts, "slm", "Show SLM (Snapshot Lifecycle Management) policies")
}

func renderSLM(ctx context.Context, es *dashboard.Elasticsearch) error {
	policies, err := slmPolicies(ctx, es)
	if err != nil {
		return err
	}

	tbl := table.New()
	tbl.Title("SLM policies")
	for _, p := range policies {
		tbl.Add(
			p.Name,
			p.Schedule,
			p.Repository,
			p.Retention,
			p.NextExecution,
			p.LastSuccess,
			p.LastFailure,
			p.SnapshotsTaken,
			p.SnapshotsFailed,
			p.SnapshotsDeleted,
			p.DeletionFailures,
		)
	}
	tbl.Column(0, table.Column{Name: "NAME", Truncate: 24})
	tbl.Column(1, table.Column{Name: "SCHEDULE", Truncate: 20})
	tbl.Column(2, table.Column{Name: "REPOSITORY", Truncate: 18})
	tbl.Column(3, table.Column{Name: "RETENTION", Truncate: 24})
	tbl.Column(4, table.Column{Name: "NEXT", Truncate: 19})
	tbl.Column(5, table.Column{Name: "SUCCESS", Truncate: 19})
	tbl.Column(6, table.Column{Name: "FAILURE", Truncate: 19})
	tbl.Column(7, table.Column{Name: "TAKEN", Alignment: table.Right, Width: 7})
	tbl.Column(8, table.Column{
		Name:      "FAILED",
		Alignment: table.Right,
		Width:     7,
		Color:     color.FgGreen,
		Colors: []table.ColorRule{
			{Condition: "> 0", Color: color.FgRed},
		},
	})
	tbl.Column(9, table.Column{Name: "DELETED", Alignment: table.Right, Width: 8})
	tbl.Column(10, table.Column{
		Name:      "DEL.FAIL",
		Alignment: table.Right,
		Width:     8,
		Color:     color.FgGreen,
		Colors: []table.ColorRule{
			{Condition: "> 0", Color: color.FgRed},
		},
	})
	tbl.Margin(table.Margin{Left: 2})
	tbl.Padding(3)
	tbl.Print()
	fmt.Println("")

	return nil
}

func slmPolicies(ctx context.Context, es *dashboard.Elasticsearch) ([]slmPolicy, error) {
	lifecycleRes, err := es.Client().SlmGetLifecycle(
		es.Client().SlmGetLifecycle.WithContext(ctx),
	)
	if err != nil {
		return nil, err
	}
	defer lifecycleRes.Body.Close()

	if lifecycleRes.IsError() {
		return nil, fmt.Errorf("error en la respuesta de _slm/policy: %s", lifecycleRes.String())
	}

	var lifecycle map[string]struct {
		Policy struct {
			Name       string `json:"name"`
			Schedule   string `json:"schedule"`
			Repository string `json:"repository"`
			Retention  *struct {
				ExpireAfter string `json:"expire_after"`
				MinCount    int    `json:"min_count"`
				MaxCount    int    `json:"max_count"`
			} `json:"retention"`
		} `json:"policy"`
		NextExecutionMillis int64 `json:"next_execution_millis"`
		LastSuccess         *struct {
			SnapshotName string `json:"snapshot_name"`
			Time         any    `json:"time"`
		} `json:"last_success"`
		LastFailure *struct {
			SnapshotName string `json:"snapshot_name"`
			Time         any    `json:"time"`
		} `json:"last_failure"`
		Stats struct {
			SnapshotsTaken           any `json:"snapshots_taken"`
			SnapshotsFailed          any `json:"snapshots_failed"`
			SnapshotsDeleted         any `json:"snapshots_deleted"`
			SnapshotDeletionFailures any `json:"snapshot_deletion_failures"`
		} `json:"stats"`
	}
	if err := json.NewDecoder(lifecycleRes.Body).Decode(&lifecycle); err != nil {
		return nil, fmt.Errorf("error al decodificar _slm/policy: %w", err)
	}

	statByPolicy, err := slmPolicyStats(ctx, es)
	if err != nil {
		return nil, err
	}

	policies := make([]slmPolicy, 0, len(lifecycle))
	for id, item := range lifecycle {
		name := item.Policy.Name
		if name == "" {
			name = id
		}

		policy := slmPolicy{
			Name:             name,
			Schedule:         item.Policy.Schedule,
			Repository:       item.Policy.Repository,
			Retention:        formatSLMRetention(item.Policy.Retention),
			NextExecution:    formatMillis(item.NextExecutionMillis),
			LastSuccess:      formatSLMInvocation(item.LastSuccess),
			LastFailure:      formatSLMInvocation(item.LastFailure),
			SnapshotsTaken:   parseIntAny(item.Stats.SnapshotsTaken),
			SnapshotsFailed:  parseIntAny(item.Stats.SnapshotsFailed),
			SnapshotsDeleted: parseIntAny(item.Stats.SnapshotsDeleted),
			DeletionFailures: parseIntAny(item.Stats.SnapshotDeletionFailures),
		}

		if stats, ok := statByPolicy[id]; ok {
			policy.SnapshotsTaken = stats.SnapshotsTaken
			policy.SnapshotsFailed = stats.SnapshotsFailed
			policy.SnapshotsDeleted = stats.SnapshotsDeleted
			policy.DeletionFailures = stats.DeletionFailures
		} else if stats, ok := statByPolicy[name]; ok {
			policy.SnapshotsTaken = stats.SnapshotsTaken
			policy.SnapshotsFailed = stats.SnapshotsFailed
			policy.SnapshotsDeleted = stats.SnapshotsDeleted
			policy.DeletionFailures = stats.DeletionFailures
		}

		policies = append(policies, policy)
	}

	return policies, nil
}

func slmPolicyStats(ctx context.Context, es *dashboard.Elasticsearch) (map[string]slmPolicy, error) {
	res, err := es.Client().SlmGetStats(
		es.Client().SlmGetStats.WithContext(ctx),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error en la respuesta de _slm/stats: %s", res.String())
	}

	var stats struct {
		PolicyStats []struct {
			Policy                   string `json:"policy"`
			SnapshotsTaken           int64  `json:"snapshots_taken"`
			SnapshotsFailed          int64  `json:"snapshots_failed"`
			SnapshotsDeleted         int64  `json:"snapshots_deleted"`
			SnapshotDeletionFailures int64  `json:"snapshot_deletion_failures"`
		} `json:"policy_stats"`
	}
	if err := json.NewDecoder(res.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("error al decodificar _slm/stats: %w", err)
	}

	byPolicy := make(map[string]slmPolicy, len(stats.PolicyStats))
	for _, stat := range stats.PolicyStats {
		byPolicy[stat.Policy] = slmPolicy{
			SnapshotsTaken:   stat.SnapshotsTaken,
			SnapshotsFailed:  stat.SnapshotsFailed,
			SnapshotsDeleted: stat.SnapshotsDeleted,
			DeletionFailures: stat.SnapshotDeletionFailures,
		}
	}

	return byPolicy, nil
}

func formatSLMRetention(retention *struct {
	ExpireAfter string `json:"expire_after"`
	MinCount    int    `json:"min_count"`
	MaxCount    int    `json:"max_count"`
}) string {
	if retention == nil {
		return "-"
	}

	parts := []string{}
	if retention.ExpireAfter != "" {
		parts = append(parts, retention.ExpireAfter)
	}
	if retention.MinCount > 0 {
		parts = append(parts, fmt.Sprintf("min:%d", retention.MinCount))
	}
	if retention.MaxCount > 0 {
		parts = append(parts, fmt.Sprintf("max:%d", retention.MaxCount))
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, " ")
}

func formatSLMInvocation(invocation *struct {
	SnapshotName string `json:"snapshot_name"`
	Time         any    `json:"time"`
}) string {
	if invocation == nil {
		return "-"
	}
	if formatted := formatMillis(parseIntAny(invocation.Time)); formatted != "-" {
		return formatted
	}
	if invocation.SnapshotName != "" {
		return invocation.SnapshotName
	}
	return "-"
}

func formatMillis(millis int64) string {
	if millis <= 0 {
		return "-"
	}
	return time.UnixMilli(millis).Format("2006-01-02 15:04:05")
}

func parseIntAny(value any) int64 {
	switch v := value.(type) {
	case float64:
		return int64(v)
	case string:
		i, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			return i
		}
	}
	return 0
}
