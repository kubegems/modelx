package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
	"kubegems.io/modelx/pkg/client"
)

func NewPullCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "pull <url> <dir>",
		Example: `
  modex pull  https://registry.example.com/repo/name@version .
		`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
			defer cancel()
			if len(args) == 0 {
				return errors.New("at least one argument is required")
			}
			if len(args) == 1 {
				args = append(args, "")
			}
			onprogress := func(status client.ProgressStatistic) {
				fmt.Printf("%s: %d/%d %s\n", status.Name, status.Count, status.Total, status.Status)
			}
			return client.Pull(ctx, args[0], args[1], onprogress)
		},
	}
	return cmd
}
