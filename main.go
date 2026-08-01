package main

import (
	"context"
	"log"

	"github.com/nicola-strappazzon/dash/cmd"
)

func main() {
	ctx := context.Background()

	if err := cmd.NewRootCommand(ctx).Execute(); err != nil {
		log.Fatal(err)
	}
}
