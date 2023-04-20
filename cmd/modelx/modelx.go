package main

import (
	"crypto/tls"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"kubegems.io/modelx/cmd/modelx/completion"
	"kubegems.io/modelx/cmd/modelx/model"
	"kubegems.io/modelx/cmd/modelx/repo"
)

const ErrExitCode = 1

func main() {
	if err := NewModelxCmd().Execute(); err != nil {
		os.Exit(ErrExitCode)
	}
}

func NewModelxCmd() *cobra.Command {
	insecureSkipVerify := false
	cmd := model.NewModelxCmd()
	cmd.AddCommand(
		repo.NewRepoCmd(),
		completion.CompletionCmd,
	)
	cmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if insecureSkipVerify {
			http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
		}
	}
	cmd.PersistentFlags().BoolVarP(&insecureSkipVerify, "insecure", "", insecureSkipVerify, "tls insecure skip verify")
	return cmd
}
