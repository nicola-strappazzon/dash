package runner

import (
	"context"
	"fmt"

	"github.com/nicola-strappazzon/dash/internal/command"
	"github.com/nicola-strappazzon/dash/internal/driver/mysql"
	"github.com/spf13/cobra"
)

type RenderFunc func(context.Context, *mysql.MySQL) error

var registry = map[string]RenderFunc{}

func Register(section string, render RenderFunc) {
	registry[section] = render
}

func SectionCommand(ctx context.Context, opts *command.Options, section, short string, render RenderFunc) *cobra.Command {
	Register(section, render)
	return &cobra.Command{
		Use:   section,
		Short: short,
		Args:  cobra.ArbitraryArgs,
		RunE: command.Repeat(ctx, opts, func(cmd *cobra.Command, args []string) error {
			return RenderSections(ctx, opts, append([]string{section}, args...))
		}),
	}
}

func RenderSections(ctx context.Context, opts *command.Options, sections []string) error {
	if err := validateSections(sections); err != nil {
		return err
	}

	return withMySQL(opts, func(db *mysql.MySQL) error {
		defer db.Close()
		for _, section := range sections {
			render := registry[section]
			if err := render(ctx, db); err != nil {
				return err
			}
		}
		return nil
	})
}

func validateSections(sections []string) error {
	seen := map[string]bool{}
	for _, section := range sections {
		if _, ok := registry[section]; !ok {
			return fmt.Errorf("unknown mysql section %q", section)
		}
		if seen[section] {
			return fmt.Errorf("duplicate mysql section %q", section)
		}
		seen[section] = true
	}
	return nil
}

func withMySQL(opts *command.Options, run func(*mysql.MySQL) error) error {
	db, err := mysql.New(mysql.Config{
		Host:               opts.MySQL.Host,
		Username:           opts.MySQL.Username,
		Password:           opts.MySQL.Password,
		Database:           opts.MySQL.Database,
		TLS:                opts.MySQL.TLS,
		InsecureSkipVerify: opts.MySQL.InsecureTLS,
	})
	if err != nil {
		return err
	}
	return run(db)
}
