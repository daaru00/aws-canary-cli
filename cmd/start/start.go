package start

import (
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/service/synthetics"
	"github.com/daaru00/aws-canary-cli/internal/aws"
	"github.com/daaru00/aws-canary-cli/internal/canary"
	"github.com/daaru00/aws-canary-cli/internal/config"
	"github.com/urfave/cli/v2"
)

// NewCommand - Return start commands
func NewCommand(globalFlags []cli.Flag) *cli.Command {
	return &cli.Command{
		Name:  "start",
		Usage: "Start a Synthetics Canary",
		Flags: append(globalFlags, []cli.Flag{
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
				Usage:   "Select all canaries",
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
	canaries, err = config.AskMultipleCanariesSelection(c, *canaries)
	if err != nil {
		return err
	}

	// Setup wait group for async jobs
	var waitGroup sync.WaitGroup
	waitGroup.Add(len(*canaries))

	// Setup deploy chan error
	errs := make(chan error)

	// Loop over found canaries
	for _, c := range *canaries {

		// Execute parallel deploy
		go func(c *canary.Canary) {
			err := SingleCanary(c)
			waitGroup.Done()

			errs <- err
			close(errs)
		}(c)
	}

	// Wait until all remove ends
	waitGroup.Wait()

	// Check remove error
	for err := range errs {
		if err != nil {
			return err
		}
	}

	return nil
}

// SingleCanary start single canary
func SingleCanary(canary *canary.Canary) error {
	// Check if deployed
	if canary.IsDeployed() == false {
		return fmt.Errorf("[%s] Error: not yet deployed", canary.Name)
	}

	// Get canary status
	currentStatus, err := canary.GetStatus()
	if err != nil {
		return err
	}

	// Check if already stopped or never started
	if *currentStatus.State == "RUNNING" {
		fmt.Println(fmt.Sprintf("[%s] Skipped: not in a startable state %s", canary.Name, *currentStatus.State))
		return nil
	}

	// Start canary
	fmt.Println(fmt.Sprintf("[%s] Starting..", canary.Name))
	err = canary.Start()
	if err != nil {
		return err
	}

	// Wait until canary ends
	fmt.Println(fmt.Sprintf("[%s] Waiting..", canary.Name))
	var status *synthetics.CanaryStatus
	for {
		time.Sleep(1000 * time.Millisecond)

		// Get canary status
		status, err = canary.GetStatus()
		if err != nil {
			return err
		}

		// Check canary state
		if status != nil && *status.State != "RUNNING" {
			time.Sleep(2000 * time.Millisecond)
			break
		}
	}

	// Get last run
	run, err := canary.GetLastRun()
	if err != nil {
		return nil
	}

	// Check for run error
	if *run.Status.State == "FAILED" {
		return fmt.Errorf("[%s] Fail: %s", canary.Name, *run.Status.StateReason)
	}

	fmt.Println(fmt.Sprintf("[%s] Passed!", canary.Name))

	return nil
}