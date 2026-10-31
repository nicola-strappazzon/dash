package command

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestWatchClearUsesAlternateScreen(t *testing.T) {
	for _, tt := range []struct {
		name, format, metric string
		clear                bool
		watch                time.Duration
		wantAlternate        bool
	}{
		{"table watch", "table", "", true, time.Millisecond, true},
		{"watch without clear", "table", "", false, time.Millisecond, false},
		{"single execution", "table", "", true, 0, false},
		{"csv watch", "csv", "", true, time.Millisecond, false},
		{"json watch", "json", "", true, time.Millisecond, false},
		{"metric watch", "table", "CPU", true, time.Millisecond, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			file, err := os.CreateTemp(t.TempDir(), "stdout")
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			stdout := os.Stdout
			os.Stdout = file
			defer func() { os.Stdout = stdout }()
			opts := &Options{Format: tt.format, Metric: tt.metric, Clear: tt.clear, Watch: tt.watch}
			ctx := WithOutput(context.Background(), opts)
			calls := 0
			failure := errors.New("stop polling")
			run := Repeat(ctx, opts, func(*cobra.Command, []string) error {
				calls++
				if calls == 2 || tt.watch == 0 {
					return failure
				}
				if tt.metric != "" {
					outputFrom(ctx).metrics["cpu"] = true
				}
				return nil
			})
			if err := run(&cobra.Command{}, nil); !errors.Is(err, failure) {
				t.Fatal(err)
			}
			data, err := os.ReadFile(file.Name())
			if err != nil {
				t.Fatal(err)
			}
			out := string(data)
			if tt.wantAlternate {
				if strings.Count(out, "\033[?1049h") != 1 || !strings.HasSuffix(out, "\033[?1049l") {
					t.Fatalf("screen not entered once and restored on error: %q", out)
				}
				if strings.Count(out, "\033[H\033[2J") != 2 {
					t.Fatalf("refreshes did not clear the alternate screen: %q", out)
				}
			} else if strings.Contains(out, "1049") {
				t.Fatalf("unexpected alternate screen: %q", out)
			}
		})
	}
}
