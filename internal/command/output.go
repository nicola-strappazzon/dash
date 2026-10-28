package command

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

type jsonOutputKey struct{}

// WithJSON shares the flag value with renderers constructed before flag parsing.
func WithJSON(ctx context.Context, enabled *bool) context.Context {
	return context.WithValue(ctx, jsonOutputKey{}, enabled)
}

func JSON(ctx context.Context) bool {
	enabled, _ := ctx.Value(jsonOutputKey{}).(*bool)
	return enabled != nil && *enabled
}

// Both go-table tables and vertical rows implement this output boundary.
type printable interface {
	Print()
	ToJSON() ([]byte, error)
}

func Print(ctx context.Context, view printable) error {
	if JSON(ctx) {
		data, err := view.ToJSON()
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(os.Stdout, string(data))
		return err
	}
	view.Print()
	fmt.Println()
	return nil
}

func PrintJSON(value any) error {
	return json.NewEncoder(os.Stdout).Encode(value)
}

// Message keeps supplementary text out of machine-readable stdout.
func Message(ctx context.Context, format string, args ...any) {
	writer := os.Stdout
	if JSON(ctx) {
		writer = os.Stderr
	}
	fmt.Fprintf(writer, format+"\n", args...)
}
