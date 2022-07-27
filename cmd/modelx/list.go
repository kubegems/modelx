package main

import (
	"context"
	"errors"
	"os"
	"os/signal"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"kubegems.io/modelx/pkg/client"
	"kubegems.io/modelx/pkg/version"
)

func NewListCmd() *cobra.Command {
	search := ""
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list <url>",
		Example: `
  modex list  https://registry.example.com --search "gpt"
  modex list  https://registry.example.com/repo/model --search "v*"
  modex list  https://registry.example.com/repo/model@v1
		`,
		Version:      version.Get().String(),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
			defer cancel()
			if len(args) == 0 {
				return errors.New("at least one argument is required")
			}
			items, err := client.List(ctx, args[0], search)
			if err != nil {
				return err
			}
			t := table.NewWriter()
			t.SetOutputMirror(os.Stdout)
			t.AppendHeader(table.Row(items.Header))
			for _, item := range items.Items {
				t.AppendRow(table.Row(item))
			}
			t.Render()
			return nil
		},
	}
	cmd.Flags().StringVar(&search, "search", search, "search")
	return cmd
}
