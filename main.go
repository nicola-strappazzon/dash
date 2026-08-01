package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nicola-strappazzon/dash/cmd"
)

func main() {
	ctx := context.Background()

	if err := cmd.NewRootCommand(ctx).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
