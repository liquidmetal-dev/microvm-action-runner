package main

import (
	"log"
	"os"

	"github.com/weaveworks-liquidmetal/microvm-action-runner/pkg/command"
)

func main() {
	app := command.NewApp(os.Stdout)

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
