package repo

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewRepoAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "add a new repository to local cache",
		Long:  "Add a repository",
		Example: `
	# Add a repository
	modelx repo add my-repo https://modelx.example.com
		`,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 1 {
				return []string{"http://", "https://"}, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
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
