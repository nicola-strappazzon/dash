package settings

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/nicola-strappazzon/dash/cmd/elasticsearch/internal/runner"
	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/nicola-strappazzon/dash/internal/driver/elasticsearch"
	"github.com/nicola-strappazzon/go-table"
	"github.com/spf13/cobra"
)

const (
	allocationPrefix = "cluster.routing.allocation."
	recoveryPrefix   = "indices.recovery."
)

func NewCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	cmd := runner.SectionCommand(ctx, opts, "settings", "Show cluster allocation and recovery-throttle settings (excludes/includes/requires, concurrency, bandwidth, watermarks)", Render)

	// Running "settings" bare (no other section to combine with, e.g.
	// "settings health") prints help instead of silently rendering — this
	// command also holds exclude-node/include-node/clear/show, so nothing
	// should happen by assumption; run "settings show" to render.
	render := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		return render(cmd, args)
	}

	cmd.AddCommand(
		newShowCommand(ctx, opts),
		newSetCommand(ctx, opts),
		newSetIndexCommand(ctx, opts),
		newExcludeNodeCommand(ctx, opts),
		newIncludeNodeCommand(ctx, opts),
		newExcludeIndexCommand(ctx, opts),
		newIncludeIndexCommand(ctx, opts),
		newClearNodeFiltersCommand(ctx, opts),
	)

	return cmd
}

func newShowCommand(ctx context.Context, opts *command.Options) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show the current allocation settings",
		Args:  cobra.NoArgs,
		RunE: command.Repeat(ctx, opts, func(cmd *cobra.Command, args []string) error {
			return runner.WithElasticsearch(opts, func(es *elasticsearch.Elasticsearch) error {
				return Render(ctx, es)
			})
		}),
	}
}

func Render(ctx context.Context, es *elasticsearch.Elasticsearch) error {
	cs, err := es.ClusterSettings(ctx)
	if err != nil {
		return err
	}

	tbl := table.New()
	tbl.Title("Allocation Settings")
	seen := map[string]bool{}
	addRows(tbl, "persistent", cs.Persistent, seen)
	addRows(tbl, "transient", cs.Transient, seen)
	// Defaults are only shown for keys with no persistent/transient override,
	// so an untouched setting (e.g. the recovery bandwidth cap) still shows up
	// without flooding the table with dozens of rows nobody changed.
	addRows(tbl, "default", cs.Defaults, seen)

	tbl.Column(0, table.Column{
		Name:  "SCOPE",
		Color: color.FgGreen,
		Colors: []table.ColorRule{
			{Condition: `== "transient"`, Color: color.FgYellow},
			{Condition: `== "default"`, Color: color.FgWhite},
		},
	})
	tbl.Column(1, table.Column{Name: "KEY", Truncate: 40})
	tbl.Column(2, table.Column{Name: "VALUE", Truncate: 40})
	tbl.Column(3, table.Column{
		Name: "TYPE",
		Colors: []table.ColorRule{
			{Condition: `== "FILTER"`, Color: color.FgRed},
			{Condition: `== "CONCURRENCY"`, Color: color.FgYellow},
			{Condition: `== "RECOVERY"`, Color: color.FgYellow},
		},
	})
	tbl.Margin(table.Margin{Left: 2})
	tbl.Padding(3)
	tbl.SetWidth(table.TerminalWidth())
	tbl.Print()
	fmt.Println("")

	return nil
}

// addRows adds one row per allocation/recovery key found in scope, sorted by
// key so the output is stable across runs. seen tracks keys already added
// under an earlier (more specific) scope, so e.g. a "default" pass skips
// anything already shown as "persistent"/"transient".
func addRows(tbl table.Table, scope string, values map[string]any, seen map[string]bool) {
	keys := make([]string, 0, len(values))
	for key := range values {
		if seen[key] {
			continue
		}
		if strings.HasPrefix(key, allocationPrefix) || strings.HasPrefix(key, recoveryPrefix) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	for _, key := range keys {
		seen[key] = true
		value := settingValueString(values[key])
		switch {
		case strings.HasPrefix(key, allocationPrefix):
			short := strings.TrimPrefix(key, allocationPrefix)
			tbl.Add(scope, short, value, classify(short))
		case strings.HasPrefix(key, recoveryPrefix):
			short := "recovery." + strings.TrimPrefix(key, recoveryPrefix)
			tbl.Add(scope, short, value, "RECOVERY")
		}
	}
}

// settingValueString renders a setting's value as a single string, since
// flat_settings only flattens nested objects — a setting whose default is a
// list (e.g. cluster.routing.allocation.awareness.attributes) still comes
// back as a JSON array, not a scalar.
func settingValueString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []any:
		parts := make([]string, len(v))
		for i, item := range v {
			parts[i] = fmt.Sprint(item)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	default:
		return fmt.Sprint(v)
	}
}

// classify labels an allocation setting so exclude/include/require filters
// (the ones that decide which nodes are eligible) stand out from throttling
// and watermark knobs.
func classify(key string) string {
	switch {
	case strings.HasPrefix(key, "exclude.") || strings.HasPrefix(key, "include.") || strings.HasPrefix(key, "require."):
		return "FILTER"
	case strings.Contains(key, "concurrent"):
		return "CONCURRENCY"
	case strings.HasPrefix(key, "disk.watermark."):
		return "WATERMARK"
	case strings.HasPrefix(key, "balance.") || key == "allow_rebalance":
		return "REBALANCE"
	case key == "enable":
		return "ENABLE"
	default:
		return "OTHER"
	}
}
