package logs

import (
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/service/synthetics"
	"github.com/daaru00/aws-canary-cli/internal/aws"
	"github.com/daaru00/aws-canary-cli/internal/config"
	"github.com/urfave/cli/v2"
)

// NewCommand - Return start commands
func NewCommand(globalFlags []cli.Flag) *cli.Command {
	return &cli.Command{
		Name:  "logs",
		Usage: "Return Synthetics Canary Run logs",
		Flags: append(globalFlags, []cli.Flag{
			&cli.BoolFlag{
				Name:    "last",
				Aliases: []string{"l"},
				Usage:   "Automatically select last canary run",
			},
			&cli.StringFlag{
				Name:    "name",
				Aliases: []string{"n"},
				Usage:   "Filter canary name",
			},
		}...),
		Action:    Action,
		ArgsUsage: "[path...]",
	}
}

// Action contain the command flow
func Action(c *cli.Context) error {
	// Create AWS session
	ses := aws.NewAwsSession(c)

	// Get canaries
	canaries, err := config.LoadCanaries(c, ses)
	if err != nil {
		return err
	}

	// Ask canaries selection
	canary, err := config.AskSingleCanarySelection(c, *canaries)
	if err != nil {
		return err
	}

	// Check if deployed
	if canary.IsDeployed() == false {
		return fmt.Errorf("Canary %s not yet deployed", canary.Name)
	}

	// Retrieve runs
	runs, err := canary.GetRuns()
	if err != nil {
		return err
	}

	// Check runs
	if len(runs) == 0 {
		return errors.New("No runs found for canary")
	}

	// Ask use to select run
	var run *synthetics.CanaryRun
	if c.Bool("last") {
		run = runs[0]
	} else {
		run, err = config.AskSingleCanaryRun(runs)
		if err != nil {
			return err
		}
	}

	// Get run log
	logs, err := canary.GetRunLogs(run)
	if err != nil {
		return err
	}

	// Print logs
	fmt.Println(*logs)

	return nil
}
