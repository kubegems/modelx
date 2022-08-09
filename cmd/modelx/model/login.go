package model

import (
	"bufio"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"github.com/spf13/cobra"
	"kubegems.io/modelx/cmd/modelx/repo"
)

func NewLoginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "login <url>",
		Example: `
  modex login https://registry.example.com
		`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
			defer cancel()
			if len(args) == 0 {
				return errors.New("at least one argument is required")
			}
			fmt.Println("please input token:")
			reader := bufio.NewReader(os.Stdin)
			token, err := reader.ReadString('\n')
			token = strings.Trim(token, "\n")
			if err != nil {
				return err
			}
			return LoginModelx(ctx, args[0], token)
		},
	}
	return cmd
}

func LoginModelx(ctx context.Context, ref string, token string) error {
	reference, err := ParseReference(ref)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/oauth", reference.Registry), nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	msg, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(string(msg))
	}

	return repo.DefaultRepoManager.Set(repo.RepoDetails{
		Name:  ref,
		URL:   reference.Registry,
		Token: base64.StdEncoding.EncodeToString([]byte(token)),
	})

}
