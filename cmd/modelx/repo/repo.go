package repo

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"kubegems.io/modelx/pkg/client"
)

func NewRepoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo",
		Short: "Repository management",
		Long:  "Repository management",
	}
	cmd.AddCommand(NewRepoAddCmd())
	cmd.AddCommand(NewRepoListCmd())
	cmd.AddCommand(NewRepoRemoveCmd())

	return cmd
}

type RepoFile struct {
	Repos []RepoDetails `json:"repos,omitempty"`
}

type RepoDetails struct {
	Name  string `json:"name,omitempty"`
	URL   string `json:"url,omitempty"`
	Token string `json:"token,omitempty"`
}

func (r RepoDetails) Client() *client.Client {
	return client.NewClient(r.URL, "Bearer "+r.Token)
}

var DefaultRepoManager = Repomanager{
	Path: func() string {
		home, err := os.UserHomeDir()
		if err != nil {
			panic(err)
		}
		return filepath.Join(home, ".modelx", "repos.json")
	}(),
}

type Repomanager struct {
	Path  string
	repos RepoFile
}

func (r *Repomanager) Set(item RepoDetails) error {
	// check url
	if _, err := url.ParseRequestURI(item.URL); err != nil {
		return fmt.Errorf("invalid url: %s", item.URL)
	}

	if err := r.load(); err != nil {
		return err
	}
	var exists bool
	for i, repo := range r.repos.Repos {
		if repo.Name == item.Name {
			r.repos.Repos[i] = item
			exists = true
			break
		}
	}
	if !exists {
		r.repos.Repos = append(r.repos.Repos, item)
	}
	return r.save()
}

func (r *Repomanager) Get(name string) (RepoDetails, error) {
	if err := r.load(); err != nil {
		return RepoDetails{}, err
	}
	for _, repo := range r.repos.Repos {
		if repo.Name == name || repo.URL == name {
			return repo, nil
		}
	}
	return RepoDetails{}, fmt.Errorf("repo %s not found", name)
}

func (r *Repomanager) Remove(name string) error {
	if err := r.load(); err != nil {
		return err
	}
	for i, repo := range r.repos.Repos {
		if repo.Name == name {
			r.repos.Repos = append(r.repos.Repos[:i], r.repos.Repos[i+1:]...)
			return r.save()
		}
	}
	return fmt.Errorf("repo %s not found", name)
}

func (r *Repomanager) List() []RepoDetails {
	if err := r.load(); err != nil {
		return []RepoDetails{}
	}
	return r.repos.Repos
}

func (r *Repomanager) load() error {
	content, err := os.ReadFile(r.Path)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(r.Path), 0o755); err != nil {
			return err
		}
		content = []byte("{}")
	}
	return json.Unmarshal(content, &r.repos)
}

func (r *Repomanager) save() error {
	content, err := json.MarshalIndent(r.repos, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.Path, content, 0o644)
}
