package command

import (
	"io"

	"github.com/urfave/cli/v2"
)

// NewApp is a builder which returns a cli.App.
func NewApp(out io.Writer) *cli.App {
	app := cli.NewApp()

	if out != nil {
		app.Writer = out
	}

	app.Name = "microvm-action-runner"
	app.Usage = "A webhook service to create GitHub action runners on MicroVMs"
	app.EnableBashCompletion = true
	app.Commands = commands()

	return app
}

func commands() []*cli.Command {
	return []*cli.Command{
		startCommand(),
	}
}
