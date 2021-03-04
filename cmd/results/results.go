package results

import (
	"errors"
	"fmt"

	"github.com/daaru00/aws-canary-cli/internal/aws"
	"github.com/daaru00/aws-canary-cli/internal/config"
	"github.com/urfave/cli/v2"
)

// NewCommand - Return start commands
func NewCommand(globalFlags []cli.Flag) *cli.Command {
	return &cli.Command{
		Name:  "results",
		Usage: "Return Synthetics Canary Runs",
		Flags: append(globalFlags, []cli.Flag{
			&cli.BoolFlag{
				Name:    "last",
				Aliases: []string{"l"},
				Usage:   "Return details about last canary run",
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
		return errors.New("No run found for canary")
	}

	// Return last detail
	if c.Bool("last") {
		fmt.Println(fmt.Sprintf("Id: %s", *runs[0].Id))
		fmt.Println(fmt.Sprintf("Status: %s", *runs[0].Status.State))
		reason := *runs[0].Status.StateReason
		if len(reason) > 0 {
			fmt.Println(fmt.Sprintf("Status Reason: %s", reason))
		}
		fmt.Println(fmt.Sprintf("Started At: %s", *runs[0].Timeline.Started))
		fmt.Println(fmt.Sprintf("Compleated At: %s", *runs[0].Timeline.Completed))
		return nil
	}

	// Print results
	fmt.Println(fmt.Sprintf("%-36s\t%-7s\t%-25s\t%-25s", "Id", "Status", "Started At", "Compleated At"))
	for _, run := range runs {
		fmt.Println(fmt.Sprintf("%-36s\t%-7s\t%-25s\t%-25s", *run.Id, *run.Status.State, *run.Timeline.Started, *run.Timeline.Completed))
	}

	return nil
}
