package main

import (
	"os"

	"github.com/spf13/cobra"
	"kubegems.io/modelx/pkg/version"
)

const ErrExitCode = 1

func main() {
	if err := NewModelxCmd().Execute(); err != nil {
		// fmt.Println(err.Error())
		os.Exit(ErrExitCode)
	}
}

func NewModelxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "modelx",
		Short:   "modelx",
		Version: version.Get().String(),
	}
	cmd.AddCommand(NewPushCmd())
	cmd.AddCommand(NewPullCmd())
	cmd.AddCommand(NewListCmd())
	return cmd
}
