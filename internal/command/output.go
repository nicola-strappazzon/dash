package command

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/nicola-strappazzon/go-table"
)

type outputKey struct{}

type output struct {
	options   *Options
	metrics   map[string]bool
	csvHeader string
}

// WithOutput shares options with renderers constructed before flag parsing.
func WithOutput(ctx context.Context, options *Options) context.Context {
	return context.WithValue(ctx, outputKey{}, &output{options: options})
}

func outputFrom(ctx context.Context) *output {
	if out, ok := ctx.Value(outputKey{}).(*output); ok {
		return out
	}
	return &output{options: &Options{Format: "table"}}
}

func JSON(ctx context.Context) bool { return outputFrom(ctx).options.Format == "json" }

// Machine reports whether stdout is reserved for unformatted data.
func Machine(ctx context.Context) bool {
	opts := outputFrom(ctx).options
	return opts.Format == "json" || opts.Format == "csv" || opts.Metric != ""
}

// Both go-table tables and vertical rows implement this output boundary.
type printable interface {
	Print()
	ToJSON() ([]byte, error)
}

type document struct {
	Title string           `json:"title"`
	Rows  []map[string]any `json:"rows"`
}

func Print(ctx context.Context, view printable) error {
	if !Machine(ctx) {
		view.Print()
		fmt.Println()
		return nil
	}
	data, err := view.ToJSON()
	if err != nil {
		return err
	}
	var doc document
	if err := decode(data, &doc); err != nil {
		return err
	}
	var columns []string
	switch v := view.(type) {
	case *Table:
		columns = v.outputColumns()
	case *table.Row:
		for i, field := range v.Fields {
			name := field.Name
			if name == "" {
				name = strconv.Itoa(i)
			}
			columns = append(columns, name)
		}
	default:
		if len(doc.Rows) > 0 {
			columns = sortedKeys(doc.Rows[0])
		}
	}
	return outputFrom(ctx).print(os.Stdout, doc, columns, time.Now())
}

// PrintData handles commands whose native result is an object rather than a
// table. Nested values use dotted paths for CSV columns and metric selection.
func PrintData(ctx context.Context, value any) error {
	out := outputFrom(ctx)
	if JSON(ctx) && out.options.Metric == "" {
		return PrintJSON(value)
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	var object any
	if err := decode(data, &object); err != nil {
		return err
	}
	fields := make(map[string]any)
	flatten(fields, "", object)
	return out.print(os.Stdout, document{Rows: []map[string]any{fields}}, sortedKeys(fields), time.Now())
}

func decode(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	return decoder.Decode(target)
}

func sortedKeys(fields map[string]any) []string {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func flatten(fields map[string]any, path string, value any) {
	switch v := value.(type) {
	case map[string]any:
		if len(v) == 0 {
			fields[path] = v
			return
		}
		for key, item := range v {
			name := key
			if path != "" {
				name = path + "." + key
			}
			flatten(fields, name, item)
		}
	case []any:
		if len(v) == 0 {
			fields[path] = v
			return
		}
		for i, item := range v {
			flatten(fields, fmt.Sprintf("%s.%d", path, i), item)
		}
	default:
		fields[path] = value
	}
}

func (out *output) print(w io.Writer, doc document, columns []string, now time.Time) error {
	opts := out.options
	metric := strings.TrimSpace(opts.Metric)
	if metric != "" {
		var selected []string
		labels := ""
		for _, column := range columns {
			switch column {
			case "METRIC", "CHECK", "VARIABLE", "STATUS", "KEY":
				if labels == "" && slices.Contains(columns, "VALUE") {
					labels = column
				}
			}
		}
		names := metricNames(metric)
		namedRows := false
		if labels != "" {
			for _, row := range doc.Rows {
				for _, name := range names {
					if strings.EqualFold(fmt.Sprint(row[labels]), name) {
						namedRows = true
					}
				}
			}
		}
		values := make(map[string]any)
		for _, name := range names {
			found := false
			if namedRows {
				for _, row := range doc.Rows {
					if strings.EqualFold(fmt.Sprint(row[labels]), name) {
						values[name] = row["VALUE"]
						selected = append(selected, name)
						found = true
						break
					}
				}
			}
			if !found && !namedRows {
				for _, column := range columns {
					if strings.EqualFold(column, name) {
						selected = append(selected, column)
						found = true
						break
					}
				}
			}
			if found {
				if out.metrics == nil {
					out.metrics = make(map[string]bool)
				}
				out.metrics[strings.ToLower(name)] = true
			}
		}
		if len(selected) == 0 {
			return nil
		} // Another section may contain the metrics.
		if len(values) > 0 {
			// Named key/value metrics become one record in the requested order.
			doc.Rows = []map[string]any{values}
		} else {
			if opts.Format == "csv" && len(columns) > 1 && !slices.Contains(selected, columns[0]) {
				selected = append([]string{columns[0]}, selected...)
			}
			for i, row := range doc.Rows {
				fields := make(map[string]any, len(selected))
				for _, column := range selected {
					fields[column] = row[column]
				}
				doc.Rows[i] = fields
			}
		}
		columns = selected
	}

	if opts.Format == "json" {
		return json.NewEncoder(w).Encode(doc)
	}
	writer := csv.NewWriter(w)
	headerColumns := columns
	if opts.Timestamp {
		headerColumns = append([]string{"timestamp"}, columns...)
	}
	if metric == "" {
		header := strings.Join(headerColumns, "\x00")
		if header != out.csvHeader {
			if err := writer.Write(headerColumns); err != nil {
				return err
			}
			out.csvHeader = header
		}
	}
	for _, row := range doc.Rows {
		values := make([]string, 0, len(headerColumns))
		if opts.Timestamp {
			values = append(values, fmt.Sprint(now.Unix()))
		}
		for _, column := range columns {
			value, err := rawString(row[column])
			if err != nil {
				return err
			}
			values = append(values, value)
		}
		// Without an explicit machine format, --metric retains plain scalar output.
		if opts.Format != "csv" && len(values) == 1 {
			if _, err := fmt.Fprintln(w, values[0]); err != nil {
				return err
			}
		} else if err := writer.Write(values); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func rawString(value any) (string, error) {
	switch v := value.(type) {
	case nil:
		return "NULL", nil
	case string:
		return v, nil
	case map[string]any, []any:
		data, err := json.Marshal(v)
		return string(data), err
	default:
		return fmt.Sprint(v), nil
	}
}

func PrintJSON(value any) error { return json.NewEncoder(os.Stdout).Encode(value) }

// Message keeps supplementary text out of machine-readable stdout.
func Message(ctx context.Context, format string, args ...any) {
	writer := os.Stdout
	if Machine(ctx) {
		writer = os.Stderr
	}
	fmt.Fprintf(writer, format+"\n", args...)
}

// metricNames preserves the requested column order and ignores duplicate names.
func metricNames(value string) []string {
	var names []string
	seen := make(map[string]bool)
	for _, part := range strings.Split(value, ",") {
		name := strings.TrimSpace(part)
		key := strings.ToLower(name)
		if name != "" && !seen[key] {
			names = append(names, name)
			seen[key] = true
		}
	}
	return names
}
