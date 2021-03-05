package main

import (
	"fmt"
	"log"
	"os"

	"github.com/daaru00/aws-canary-cli/cmd/build"
	"github.com/daaru00/aws-canary-cli/cmd/deploy"
	"github.com/daaru00/aws-canary-cli/cmd/logs"
	"github.com/daaru00/aws-canary-cli/cmd/remove"
	"github.com/daaru00/aws-canary-cli/cmd/results"
	"github.com/daaru00/aws-canary-cli/cmd/start"
	"github.com/daaru00/aws-canary-cli/cmd/stop"
	"github.com/daaru00/aws-canary-cli/internal/config"
	"github.com/urfave/cli/v2"
)

func main() {
	var err error

	// Load .env
	err = config.LoadDotEnv()
	if err != nil {
		log.Fatal(err)
	}

	// Setup global flags
	globalFlags := []cli.Flag{
		&cli.StringFlag{
			Name:    "profile",
			Aliases: []string{"p"},
			Usage:   "AWS profile name",
			EnvVars: []string{"AWS_PROFILE", "AWS_DEFAULT_PROFILE"},
		},
		&cli.StringFlag{
			Name:    "region",
			Aliases: []string{"r"},
			Usage:   "AWS region",
			EnvVars: []string{"AWS_REGION", "AWS_DEFAULT_REGION"},
		},
		&cli.StringFlag{
			Name:    "config-file",
			Aliases: []string{"cf"},
			Usage:   "Config file name",
			Value:   "canary.yml",
			EnvVars: []string{"CANARY_CONFIG_FILE"},
		},
		&cli.StringFlag{
			Name:    "config-parser",
			Aliases: []string{"cp"},
			Usage:   "Config file parser, valid values are \"yml\" or \"json\"",
			Value:   "yml",
			EnvVars: []string{"CANARY_CONFIG_PARSER"},
		},
	}

	// Create CLI application
	app := &cli.App{
		Name:        "aws-canary",
		Description: "AWS Synthetics Canary Helper CLI",
		Usage:       "Deploy and manage AWS Synthetics Canaries",
		UsageText:   "./aws-canary [global options] command [command options] [path...]",
		Version:     "VERSION", // this will be overridden during build phase
		Commands: []*cli.Command{
			build.NewCommand(globalFlags),
			deploy.NewCommand(globalFlags),
			remove.NewCommand(globalFlags),
			start.NewCommand(globalFlags),
			stop.NewCommand(globalFlags),
			logs.NewCommand(globalFlags),
			results.NewCommand(globalFlags),
		},
		Flags:                globalFlags,
		EnableBashCompletion: true,
		Before: func(c *cli.Context) error {
			if len(c.String("profile")) > 0 {
				os.Setenv("AWS_PROFILE", c.String("profile"))
			}
			if len(c.String("region")) > 0 {
				os.Setenv("AWS_REGION", c.String("region"))
			}
			if len(c.String("config-file")) > 0 {
				os.Setenv("CONFIG_FILE", c.String("config-file"))
			}
			if len(c.String("config-parser")) > 0 {
				os.Setenv("CONFIG_PARSER", c.String("config-parser"))
			}
			return nil
		},
	}

	// Run the CLI application
	err = app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
