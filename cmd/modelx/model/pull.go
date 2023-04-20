package model

import (
	"context"
	"errors"
	"fmt"
	"path"

	"github.com/spf13/cobra"
	"kubegems.io/modelx/cmd/modelx/repo"
)

func NewPullCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "pull a model from a repository",
		Example: `
	# Pull project/demo version latest to dirctory demo by default

  		modex pull  https://myrepo/project/demo

	# Pull project/demo to current dirctoty

		modex pull  https://myrepo/project/demo@version .
		
	# Pull project/demo to dirctoty abc

		modex pull  https://myrepo/project/demo@version abc

		`,
		SilenceUsage: true,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return repo.CompleteRegistryRepositoryVersion(toComplete)
			}
			if len(args) == 1 {
				return nil, cobra.ShellCompDirectiveFilterDirs
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := BaseContext()
			defer cancel()
			if len(args) == 0 {
				return errors.New("at least one argument is required")
			}
			if len(args) == 1 {
				args = append(args, "")
			}
			return PullModelx(ctx, args[0], args[1])
		},
	}
	return cmd
}

func PullModelx(ctx context.Context, ref string, into string) error {
	reference, err := ParseReference(ref)
	if err != nil {
		return err
	}
	if reference.Repository == "" {
		return errors.New("repository is not specified")
	}
	if into == "" {
		into = path.Base(reference.Repository)
	}
	fmt.Printf("Pulling %s into %s \n", reference.String(), into)
	return reference.Client().Pull(ctx, reference.Repository, reference.Version, into)
}
