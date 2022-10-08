package repo

import (
	"context"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

const (
	SplitorRepo    = "/"
	SplitorVersion = "@"
)

func NewRepoListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list local repositories",
		Long:  "List repositories",
		Example: `
	# List repositories

		modelx repo list

		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			details := DefaultRepoManager.List()
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

func CompleteRegistryRepositoryVersion(toComplete string) ([]string, cobra.ShellCompDirective) {
	if i := strings.Index(toComplete, SplitorRepo); i != -1 {
		registry, repositoryToComplete := toComplete[:i], toComplete[i+1:]
		if i := strings.Index(repositoryToComplete, SplitorVersion); i != -1 {
			repository, versionToComplete := repositoryToComplete[:i], repositoryToComplete[i+1:]
			return CompleteVersion(registry, repository, versionToComplete)
		}
		completes, d := CompleteRepositories(registry, repositoryToComplete)
		if repositoryToComplete != "" {
			completes = append(completes, registry+SplitorRepo+repositoryToComplete+SplitorVersion)
		}
		return completes, d
	}
	completes, d := CompleteRegistry(toComplete)
	if toComplete != "" {
		completes = append(completes, toComplete+SplitorRepo)
	}
	return completes, d
}

func CompleteRegistry(toComplete string) ([]string, cobra.ShellCompDirective) {
	names := []string{}
	for _, item := range DefaultRepoManager.List() {
		if toComplete != "" {
			if strings.HasPrefix(item.Name, toComplete) {
				names = append(names, item.Name)
			}
		} else {
			names = append(names, item.Name)
		}
	}
	return names, cobra.ShellCompDirectiveNoSpace
}

func CompleteRepositories(registry string, repositoryToComplete string) ([]string, cobra.ShellCompDirective) {
	details, err := DefaultRepoManager.Get(registry)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	index, err := details.Client().GetGlobalIndex(context.Background(), repositoryToComplete)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	registries := []string{}
	for _, item := range index.Manifests {
		registries = append(registries, registry+SplitorRepo+item.Name)
	}
	return registries, cobra.ShellCompDirectiveNoSpace
}

func CompleteVersion(registry, repository, versionToComplete string) ([]string, cobra.ShellCompDirective) {
	details, err := DefaultRepoManager.Get(registry)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	index, err := details.Client().GetIndex(context.Background(), repository, versionToComplete)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	versions := []string{}
	for _, item := range index.Manifests {
		versions = append(versions, registry+SplitorRepo+repository+SplitorVersion+item.Name)
	}
	return versions, cobra.ShellCompDirectiveNoSpace
}
