package repo

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewRepoRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "remove a repository",
		Long:  "Remove a repository",
		Example: `
	# Remove a repository

		modelx repo remove my-repo`,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return CompleteRegistry(toComplete)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("repo remove requires at least one argument")
			}
			for _, name := range args {
				if err := DefaultRepoManager.Remove(name); err != nil {
					return err
				}
			}
			return nil
		},
	}
	return cmd
}
