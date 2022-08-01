package repo

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewRepoAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a repository",
		Long:  "Add a repository",
		Example: `
	# Add a repository
	modelx repo add my-repo https://modelx.example.com
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return fmt.Errorf("repo add requires two arguments")
			}
			name := args[0]
			url := args[1]

			return DefaultRepoManager.Set(RepoDetails{
				Name: name,
				URL:  url,
			})
		},
	}
	return cmd
}
