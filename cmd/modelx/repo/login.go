package repo

import "github.com/spf13/cobra"

func NewRepoLoginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login to a repository",
		Long:  "Login to a repository",
	}
	return cmd
}
