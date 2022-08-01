package model

import (
	"context"
	"errors"
	"os"
	"os/signal"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"kubegems.io/modelx/pkg/client"
	"kubegems.io/modelx/pkg/types"
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
	switch {
	case reference.Repository == "" && reference.Version == "":
		// list repositories
		index, err := client.GetIndex(ctx, reference, search)
		if err != nil {
			return nil, err
		}
		show := &ShowList{
			Header: []any{"Name", "URL", "Description"},
		}

		for _, item := range index.Manifests {
			ref := client.Reference{Registry: reference.Registry, Repository: item.Name}
			show.Items = append(show.Items, []any{item.Name, ref.String(), item.Annotations[AnnotationDescription]})
		}
		return show, nil
	case reference.Repository != "" && reference.Version != "":
		// list files
		manifest, err := client.GetManifest(ctx, reference)
		if err != nil {
			return nil, err
		}
		show := &ShowList{
			Header: []any{"File", "Size", "Digest", "Modified"},
		}
		items := append([]types.Descriptor{manifest.Config}, manifest.Blobs...)
		for _, item := range items {
			show.Items = append(show.Items, []any{item.Name, item.Size, item.Digest, item.Modified})
		}
		return show, nil
	case reference.Repository != "" && reference.Version == "":
		// list versions
		index, err := client.GetIndex(ctx, reference, search)
		if err != nil {
			return nil, err
		}
		show := &ShowList{
			Header: []any{"Version", "URL", "Description"},
		}
		for _, item := range index.Manifests {
			ref := client.Reference{Registry: reference.Registry, Repository: reference.Repository, Version: item.Name}
			show.Items = append(show.Items, []any{item.Name, ref.String(), item.Annotations[AnnotationDescription]})
		}
		return show, nil
	default:
		return nil, errors.New("invalid reference")
	}
}
