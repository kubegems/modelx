package repo

import (
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func NewRepoListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List repositories",
		Long:  "List repositories",
		Example: `
		# List repositories
		modelx repo list
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			details, err := DefaultRepoManager.List()
			if err != nil {
				return err
			}
			t := table.NewWriter()
			t.SetOutputMirror(cmd.OutOrStdout())
			t.AppendHeader(table.Row{"Name", "URL"})
			for _, item := range details {
				t.AppendRow(table.Row{item.Name, item.URL})
			}
			t.Render()
			return nil
		},
	}
	return cmd
}
