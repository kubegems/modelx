package model

import (
	"github.com/spf13/cobra"
	"kubegems.io/modelx/pkg/version"
)

func NewModelxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "modelx",
		Short:   "modelx",
		Version: version.Get().String(),
	}
	cmd.AddCommand(NewInitCmd())
	cmd.AddCommand(NewLoginCmd())
	cmd.AddCommand(NewListCmd())
	cmd.AddCommand(NewInfoCmd())
	cmd.AddCommand(NewPushCmd())
	cmd.AddCommand(NewPullCmd())
	return cmd
}
