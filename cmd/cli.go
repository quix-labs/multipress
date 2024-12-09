package cmd

import (
	"github.com/quix-labs/multipress/cmd/backup"
	"github.com/quix-labs/multipress/cmd/deploy"
	"github.com/quix-labs/multipress/cmd/doctor"
	"github.com/quix-labs/multipress/cmd/down"
	newcmd "github.com/quix-labs/multipress/cmd/new"
	"github.com/quix-labs/multipress/cmd/replicate"
	"github.com/quix-labs/multipress/cmd/up"
	"github.com/urfave/cli/v2"
	"os"
)

func Run() error {
	app := cli.App{
		Usage: "Generate and replicate Wordpress onto multiple instances",
		Commands: []*cli.Command{
			backup.Command(),
			down.Command(),
			up.Command(),
			deploy.Command(),
			doctor.Command(),
			newcmd.Command(),
			replicate.Command(),
		},
	}

	return app.Run(os.Args)
}
