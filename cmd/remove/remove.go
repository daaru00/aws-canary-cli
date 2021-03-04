package remove

import (
	"errors"
	"fmt"
	"sync"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/daaru00/aws-canary-cli/cmd/stop"
	"github.com/daaru00/aws-canary-cli/internal/aws"
	"github.com/daaru00/aws-canary-cli/internal/bucket"
	"github.com/daaru00/aws-canary-cli/internal/canary"
	"github.com/daaru00/aws-canary-cli/internal/config"
	"github.com/daaru00/aws-canary-cli/internal/iam"
	"github.com/urfave/cli/v2"
)

// NewCommand - Return remove commands
func NewCommand(globalFlags []cli.Flag) *cli.Command {
	return &cli.Command{
		Name:    "remove",
		Aliases: []string{"delete"},
		Usage:   "Remove a Synthetics Canary",
		Flags: append(globalFlags, []cli.Flag{
			&cli.StringFlag{
				Name:    "artifact-bucket",
				Usage:   "The Artifact bucket name",
				EnvVars: []string{"CANARY_ARTIFACT_BUCKET", "CANARY_ARTIFACT_BUCKET_NAME"},
			},
			&cli.BoolFlag{
				Name:    "bucket",
				Aliases: []string{"b"},
				Usage:   "Remove also artifact bucket",
			},
			&cli.BoolFlag{
				Name:    "yes",
				Aliases: []string{"y"},
				Usage:   "Answer yes for all confirmations",
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

	// Ask confirmation
	if c.Bool("yes") == false {
		confirm := false
		prompt := &survey.Confirm{
			Message: fmt.Sprintf("Are you sure you want to remove %d canaries?", len(*canaries)),
		}
		survey.AskOne(prompt, &confirm)

		// Check respose
		if confirm == false {
			return errors.New("Not confirmed canaries remove, skip operation")
		}
	}

	// Get caller infos
	accountID := aws.GetCallerAccountID(ses)
	region := aws.GetCallerRegion(ses)
	if accountID == nil {
		return errors.New("No valid AWS credentials found")
	}

	// Remove artifact bucket
	if c.Bool("bucket") {
		artifactBucketName := c.String("artifact-bucket")
		if len(artifactBucketName) == 0 {
			artifactBucketName = fmt.Sprintf("canary-artifact-%s-%s", *accountID, *region)
		}
		err = removeArtifactBucket(ses, &artifactBucketName)
		if err != nil {
			return err
		}
	}

	// Setup wait group for async jobs
	var waitGroup sync.WaitGroup
	waitGroup.Add(len(*canaries))

	// Setup deploy chan error
	errs := make(chan error)

	// Loop over found canaries
	for _, cy := range *canaries {

		// Execute parallel deploy
		go func(canary *canary.Canary) {
			var err error

			if canary.IsDeployed() {
				err = stop.SingleCanary(canary)
			}

			if err == nil {
				err = removeSingleCanary(ses, canary, region)
			}

			waitGroup.Done()

			errs <- err
			close(errs)
		}(cy)
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

func removeArtifactBucket(ses *session.Session, artifactBucketName *string) error {
	artifactBucket := bucket.New(ses, artifactBucketName)

	// Empty bucket
	fmt.Println(fmt.Sprintf("Empty artifact bucket.."))
	err := artifactBucket.Empty()
	if err != nil {
		return err
	}

	// Remove artifact bucket
	fmt.Println(fmt.Sprintf("Removing artifact bucket.."))
	err = artifactBucket.Remove()
	if err != nil {
		return err
	}

	return nil
}

func removeIamRole(ses *session.Session, canary *canary.Canary, roleName *string, policyName *string) error {
	role := iam.NewRole(ses, roleName)
	policy := iam.NewPolicy(ses, policyName)
	role.SetInlinePolicy(policy)

	// Check if role is deployed
	if role.IsDeployed() == false {
		return nil
	}

	// Remove role
	fmt.Println(fmt.Sprintf("[%s] Removing role..", canary.Name))
	err := role.Remove()
	if err != nil {
		return err
	}

	return nil
}

func removeSingleCanary(ses *session.Session, canary *canary.Canary, region *string) error {
	var err error

	if canary.IsDeployed() {
		// Remove canary
		fmt.Println(fmt.Sprintf("[%s] Removing..", canary.Name))
		err = canary.Remove()
		if err != nil {
			return err
		}
	}

	// Remove role
	roleName := fmt.Sprintf("CloudWatchSyntheticsRole-%s-%s", *region, canary.Name)
	policyName := fmt.Sprintf("CloudWatchSyntheticsPolicy-%s-%s", *region, canary.Name)
	err = removeIamRole(ses, canary, &roleName, &policyName)
	if err != nil {
		return err
	}

	fmt.Println(fmt.Sprintf("[%s] Remove completed!", canary.Name))
	return nil
}
