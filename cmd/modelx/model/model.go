package model

import (
	"strings"

	"github.com/spf13/cobra"
	"kubegems.io/modelx/cmd/modelx/repo"
	"kubegems.io/modelx/pkg/client"
	"kubegems.io/modelx/pkg/version"
)

func NewModelxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "modelx",
		Short:   "modelx",
		Version: version.Get().String(),
	}
	cmd.AddCommand(NewInitCmd())
	cmd.AddCommand(NewListCmd())
	cmd.AddCommand(NewInfoCmd())
	cmd.AddCommand(NewPushCmd())
	cmd.AddCommand(NewPullCmd())
	cmd.AddCommand(NewLoginCmd())
	return cmd
}

func ParseReference(ref string) (client.Reference, error) {
	if !strings.Contains(ref, "://") {
		splits := strings.SplitN(ref, ":", 2)
		details, err := repo.DefaultRepoManager.Get(splits[0])
		if err != nil {
			return client.Reference{}, err
		}
		if len(splits) == 2 {
			ref = details.URL + "/" + splits[1]
		} else {
			ref = details.URL
		}
	}
	return client.ParseReference(ref)
}
