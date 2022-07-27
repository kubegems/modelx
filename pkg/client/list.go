package client

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"kubegems.io/modelx/pkg/types"
)

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
		return ListRepositories(ctx, reference, search)
	case reference.Repository != "" && reference.Version != "":
		return ListFiles(ctx, reference)
	case reference.Repository != "" && reference.Version == "":
		return ListVersions(ctx, reference, search)
	default:
		return nil, errors.New("invalid reference")
	}
}

func ListVersions(ctx context.Context, reference Reference, search string) (*ShowList, error) {
	remote := RegistryClient{
		Client: &http.Client{},
		Addr:   reference.Registry,
	}
	index, err := remote.GetIndex(ctx, reference.Repository, search)
	if err != nil {
		return nil, err
	}

	show := &ShowList{
		Header: []any{"Version", "URL", "Description"},
		Items:  make([][]any, len(index.Manifests)),
	}
	for i, manifest := range index.Manifests {
		ref := Reference{Registry: reference.Registry, Repository: reference.Repository, Version: manifest.Name}
		show.Items[i] = []any{manifest.Name, ref.String(), manifest.Annotations[types.AnnotationDescription]}
	}
	return show, nil
}

func ListRepositories(ctx context.Context, reference Reference, search string) (*ShowList, error) {
	remote := RegistryClient{
		Client: &http.Client{},
		Addr:   reference.Registry,
	}
	index, err := remote.GetGlobalIndex(ctx, search)
	if err != nil {
		return nil, err
	}

	show := &ShowList{
		Header: []any{"Repository", "URL", "Description"},
		Items:  make([][]any, len(index.Indexes)),
	}
	for i, repo := range index.Indexes {
		ref := Reference{Registry: reference.Registry, Repository: repo.Name}
		show.Items[i] = []any{repo.Name, ref.String(), repo.Annotations[types.AnnotationDescription]}
	}
	return show, nil
}

func ListFiles(ctx context.Context, reference Reference) (*ShowList, error) {
	remote := RegistryClient{
		Client: &http.Client{},
		Addr:   reference.Registry,
	}
	index, err := remote.GetManifest(ctx, reference.Repository, reference.Version)
	if err != nil {
		return nil, err
	}

	show := &ShowList{
		Header: []any{"Name", "Digest", "Size"},
		Items:  make([][]any, len(index.Blobs)),
	}
	for i, blob := range index.Blobs {
		show.Items[i] = []any{blob.Name, blob.Digest.String(), strconv.FormatInt(blob.Size, 10)}
	}
	return show, nil
}
