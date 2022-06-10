package main

import (
	"log"
	"os"

	lib "github.com/pcyman/deplog/lib"
	"github.com/urfave/cli/v2"
)

func main() {
	var follow bool
	var container string
	var count int

	app := &cli.App{
		Name:    "deplog",
		Version: "v0.3.0",
		Authors: []*cli.Author{
			{
				Name:  "Paweł Cyman",
				Email: "pawel@cyman.xyz",
			},
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "follow",
				Aliases:     []string{"f"},
				Value:       false,
				Usage:       "follow the logs",
				Destination: &follow,
			},
			&cli.StringFlag{
				Name:        "container",
				Aliases:     []string{"c"},
				Usage:       "specify which container",
				Destination: &container,
			},
			&cli.IntFlag{
				Name:        "count",
				Aliases:     []string{"n"},
				Usage:       "how many logs per pod to query",
				Destination: &count,
			},
		},
		Usage: "dep your logs",
		Action: func(c *cli.Context) error {
			countSet := c.IsSet("count")
			containerSet := c.IsSet("container")

			deployment := c.Args().Get(0)
			if deployment == "" {
				log.Fatal("Provide a deployment")
			}

			lib.GetLogs(deployment, container, containerSet, follow, int64(count), countSet)
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
