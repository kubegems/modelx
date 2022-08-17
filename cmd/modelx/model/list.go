package model

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"kubegems.io/modelx/cmd/modelx/repo"
	"kubegems.io/modelx/pkg/client"
	"kubegems.io/modelx/pkg/client/units"
	"kubegems.io/modelx/pkg/types"
	"kubegems.io/modelx/pkg/version"
)

func NewListCmd() *cobra.Command {
	search := ""
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list manifests",
		Example: `
  modex list  https://registry.example.com --search "gpt"
  modex list  https://registry.example.com/repo/model --search "v*"
  modex list  https://registry.example.com/repo/model@v1
		`,
		Version:      version.Get().String(),
		SilenceUsage: true,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return repo.CompleteRegistryRepositoryVersion(toComplete)
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
			defer cancel()
			if len(args) == 0 {
				return errors.New("at least one argument is required")
			}
			items, err := List(ctx, args[0], search)
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

type ShowList struct {
	Header []any
	Items  [][]any
}

func List(ctx context.Context, ref string, search string) (*ShowList, error) {
	reference, err := ParseReference(ref)
	if err != nil {
		return nil, err
	}

	cli := reference.Client()
	repo, version := reference.Repository, reference.Version

	switch {
	case repo == "" && version == "":
		// list repositories
		index, err := cli.GetGlobalIndex(ctx, search)
		if err != nil {
			return nil, err
		}
		show := &ShowList{
			Header: []any{"Name", "URL"},
		}
		for _, item := range index.Manifests {
			ref := Reference{Registry: reference.Registry, Repository: item.Name}
			show.Items = append(show.Items, []any{item.Name, ref.String()})
		}
		return show, nil
	case repo != "" && version != "":
		// list files
		manifest, err := cli.GetManifest(ctx, repo, version)
		if err != nil {
			return nil, err
		}
		show := &ShowList{
			Header: []any{"File", "Type", "Size", "Digest", "Modified"},
		}
		getType := func(mt string) string {
			switch mt {
			case client.MediaTypeModelDirectoryTarGz:
				return "directory"
			case client.MediaTypeModelFile:
				return "file"
			case client.MediaTypeModelConfigYaml:
				return "config"
			default:
				return mt
			}
		}
		formattime := func(tm time.Time) string {
			return tm.Format(time.RFC3339)
		}
		items := append([]types.Descriptor{manifest.Config}, manifest.Blobs...)
		for _, item := range items {
			show.Items = append(show.Items, []any{
				item.Name,
				getType(item.MediaType),
				units.HumanSize(float64(item.Size)),
				item.Digest.Encoded()[:16],
				formattime(item.Modified),
			})
		}
		return show, nil
	case repo != "" && version == "":
		// list versions
		index, err := cli.GetIndex(ctx, repo, search)
		if err != nil {
			return nil, err
		}
		show := &ShowList{
			Header: []any{"Version", "URL", "Size"},
		}
		for _, item := range index.Manifests {
			ref := Reference{Registry: reference.Registry, Repository: repo, Version: item.Name}
			show.Items = append(show.Items, []any{item.Name, ref.String(), units.HumanSize(float64(item.Size))})
		}
		return show, nil
	default:
		return nil, errors.New("invalid reference")
	}
}
