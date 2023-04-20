package model

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
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

func BaseContext() (context.Context, context.CancelFunc) {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	if os.Getenv("DEBUG") == "1" {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		ctx = logr.NewContext(ctx, stdr.NewWithOptions(log.Default(), stdr.Options{LogCaller: stdr.Error}))
	}
	return ctx, cancel
}
