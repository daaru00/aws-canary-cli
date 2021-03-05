package build

import (
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/daaru00/aws-canary-cli/internal/aws"
	"github.com/daaru00/aws-canary-cli/internal/canary"
	"github.com/daaru00/aws-canary-cli/internal/config"
	"github.com/urfave/cli/v2"
)

// NewCommand - Return deploy commands
func NewCommand(globalFlags []cli.Flag) *cli.Command {
	return &cli.Command{
		Name:  "build",
		Usage: "Build Synthetics Canary code",
		Flags: append(globalFlags, []cli.Flag{
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
				Usage:   "Select all canaries",
			},
			&cli.BoolFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Print build command output",
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

	// Setup deploy chan error
	errs := make(chan error, len(*canaries))

	// Loop over found canaries
	for _, cy := range *canaries {

		// Execute parallel deploy
		waitGroup.Add(1)
		go func(canary *canary.Canary) {
			defer waitGroup.Done()
			output, err := SingleCanary(ses, canary)

			// Check output flag
			if c.Bool("output") && len(*output) > 0 {
				fmt.Println(fmt.Sprintf("[%s] Output: \n%s", canary.Name, *output))
			}

			errs <- err
		}(cy)
	}

	// Wait until all remove ends
	waitGroup.Wait()

	// Close errors channel
	close(errs)

	// Check errors
	for i := 0; i < len(*canaries); i++ {
		err := <-errs
		if err != nil {
			return err
		}
	}

	return nil
}

// SingleCanary build single canary code
func SingleCanary(ses *session.Session, canary *canary.Canary) (*string, error) {
	var err error
	var output string

	// Install code dependencies
	if canary.IsPythonRuntime() {
		fmt.Println(fmt.Sprintf("[%s] Installing pip dependencies..", canary.Name))
		output, err = canary.Code.InstallPipDependencies()
	} else if canary.IsNodeRuntime() {
		fmt.Println(fmt.Sprintf("[%s] Installing npm dependencies..", canary.Name))
		output, err = canary.Code.InstallNpmDependencies()
	}
	if err != nil {
		return &output, err
	}

	fmt.Println(fmt.Sprintf("[%s] Dependencies installed!", canary.Name))
	return &output, nil
}
