package main

import (
	"context"
	"errors"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
	"kubegems.io/modelx/pkg/client"
)

func NewPushCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push",
		Short: "push <url> <dir>",
		Example: `
  modex push  https://registry.example.com/repo/name@version .
		`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
			defer cancel()
			if len(args) == 0 {
				return errors.New("at least one argument is required")
			}
			if len(args) == 1 {
				args = append(args, ".")
			}
			if err := client.Push(ctx, args[0], args[1]); err != nil {
				return err
			}
			return nil
		},
	}
	return cmd
}
