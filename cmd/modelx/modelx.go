package main

import (
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
	cmd := model.NewModelxCmd()
	cmd.AddCommand(
		repo.NewRepoCmd(),
		completion.CompletionCmd,
	)
	return cmd
}
