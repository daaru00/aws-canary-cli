package deploy

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/synthetics"
	"github.com/daaru00/aws-canary-cli/cmd/build"
	"github.com/daaru00/aws-canary-cli/cmd/start"
	"github.com/daaru00/aws-canary-cli/internal/aws"
	"github.com/daaru00/aws-canary-cli/internal/bucket"
	"github.com/daaru00/aws-canary-cli/internal/canary"
	"github.com/daaru00/aws-canary-cli/internal/config"
	"github.com/daaru00/aws-canary-cli/internal/iam"
	"github.com/urfave/cli/v2"
)

// NewCommand - Return deploy commands
func NewCommand(globalFlags []cli.Flag) *cli.Command {
	return &cli.Command{
		Name:  "deploy",
		Usage: "Deploy a Synthetics Canary",
		Flags: append(globalFlags, []cli.Flag{
			&cli.StringFlag{
				Name:    "artifact-bucket",
				Usage:   "Then Artifact bucket name",
				EnvVars: []string{"CANARY_ARTIFACT_BUCKET", "CANARY_ARTIFACT_BUCKET_NAME"},
			},
			&cli.BoolFlag{
				Name:    "yes",
				Aliases: []string{"y"},
				Usage:   "Answer yes for all confirmations",
			},
			&cli.BoolFlag{
				Name:    "start",
				Aliases: []string{"s"},
				Usage:   "Start canary after deploy",
			},
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

	// Get caller infos
	accountID := aws.GetCallerAccountID(ses)
	region := aws.GetCallerRegion(ses)
	if accountID == nil {
		return errors.New("No valid AWS credentials found")
	}

	// Deploy artifact bucket
	artifactBucketName := c.String("artifact-bucket")
	if len(artifactBucketName) == 0 {
		artifactBucketName = fmt.Sprintf("cw-syn-results-%s-%s", *accountID, *region)
	}
	artifactBucket, err := deployArtifactBucket(ses, &artifactBucketName)
	if err != nil {
		return err
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
			err := deploySingleCanary(ses, region, accountID, canary, artifactBucket)
			if err == nil && c.Bool("start") {
				err = start.SingleCanary(canary)
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

func deployArtifactBucket(ses *session.Session, artifactBucketName *string) (*bucket.Bucket, error) {
	fmt.Println(fmt.Sprintf("Checking artifact bucket.."))

	// Check artifact bucket
	artifactBucket := bucket.New(ses, artifactBucketName)
	if artifactBucket.IsDeployed() == false {
		// Ask for deploy
		confirm := false
		prompt := &survey.Confirm{
			Message: fmt.Sprintf("Artifact bucket %s not found, do you want to deploy it now?", *artifactBucketName),
		}
		survey.AskOne(prompt, &confirm)

		// Check respose
		if confirm == false {
			return nil, fmt.Errorf("Artifact bucket %s not found", *artifactBucketName)
		}

		// Deploy artifact bucket
		fmt.Println(fmt.Sprintf("Deploying artifact bucket %s..", *artifactBucketName))
		err := artifactBucket.Deploy()
		if err != nil {
			return artifactBucket, err
		}
	}

	return artifactBucket, nil
}

func deployIamRole(ses *session.Session, roleName *string, policy *iam.Policy) (*iam.Role, error) {
	// Prepare role
	role := iam.NewRole(ses, roleName)
	role.SetInlinePolicy(policy)

	// Deploy role
	err := role.Deploy()
	if err != nil {
		return role, err
	}

	return role, nil
}

func buildIamPolicy(ses *session.Session, policyName *string, artifactBucket *bucket.Bucket, policyStatements *[]iam.StatementEntry, region *string, accountID *string) (*iam.Policy, error) {
	// Build policy
	policy := iam.NewPolicy(ses, policyName)
	policy.AddArtifactBucketPermission(artifactBucket)
	policy.AddLogPermission(region, accountID)
	policy.AddMetricsPermission()
	policy.AddSSMParamersPermission(region, accountID)
	policy.AddXRayPermission()

	// Add custom policy statements
	for _, statement := range *policyStatements {
		policy.AddStatement(statement)
	}

	// Return policy
	return policy, nil
}

func deploySingleCanary(ses *session.Session, region *string, accountID *string, canary *canary.Canary, artifactBucket *bucket.Bucket) error {
	var err error
	var role *iam.Role

	// Check provided role
	if len(canary.RoleName) > 0 {
		role = iam.NewRole(ses, &canary.RoleName)
	} else {

		// Deploy iam policy
		fmt.Println(fmt.Sprintf("[%s] Build policy..", canary.Name))
		policyName := fmt.Sprintf("CloudWatchSyntheticsPolicy-%s-%s", *region, canary.Name)
		policy, err := buildIamPolicy(ses, &policyName, artifactBucket, &canary.PolicyStatements, region, accountID)
		if err != nil {
			return err
		}

		// Deploy iam role
		fmt.Println(fmt.Sprintf("[%s] Deploying role..", canary.Name))
		roleName := fmt.Sprintf("CloudWatchSyntheticsRole-%s-%s", *region, canary.Name)
		role, err = deployIamRole(ses, &roleName, policy)
		if err != nil {
			return err
		}
	}

	// Elaborate path prefix
	codePathPrefix := ""
	if canary.IsPythonRuntime() {
		codePathPrefix = "python"
	} else if canary.IsNodeRuntime() {
		codePathPrefix = "nodejs/node_modules"
	}

	// Install code dependencies
	_, err = build.SingleCanary(ses, canary)
	if err != nil {
		return err
	}

	// Prepare canary code
	fmt.Println(fmt.Sprintf("[%s] Preparing code..", canary.Name))
	err = canary.Code.CreateArchive(&canary.Name, &codePathPrefix)
	if err != nil {
		return err
	}

	// Clean archive at the end of deploy
	defer cleanTemporaryResources(canary)

	// // Upload canary code
	// fmt.Println(fmt.Sprintf("[%s] Upload code..", canary.Name))
	// err = canary.Code.Upload(artifactBucket.Name)
	// if err != nil {
	// 	return err
	// }

	// Deploy canary
	fmt.Println(fmt.Sprintf("[%s] Deploying..", canary.Name))
	artifactBucketLocation := *artifactBucket.Location + "/canary/" + canary.Name
	err = canary.Deploy(role, &artifactBucketLocation)
	if err != nil {
		return err
	}

	// Wait until canary is created
	var status *synthetics.CanaryStatus
	fmt.Println(fmt.Sprintf("[%s] Waiting..", canary.Name))
	for {
		time.Sleep(1000 * time.Millisecond)

		// Get canary status
		status, err = canary.GetStatus()
		if err != nil {
			return err
		}

		// Check canary state
		if *status.State != "CREATING" && *status.State != "UPDATING" {
			break
		}
	}

	// Check for deploy error
	if *status.State == "ERROR" {
		return fmt.Errorf("[%s] Error: %s", canary.Name, *status.StateReason)
	}

	fmt.Println(fmt.Sprintf("[%s] Deploy completed!", canary.Name))
	return nil
}

func cleanTemporaryResources(canary *canary.Canary) {
	// Clean temporary resources
	fmt.Println(fmt.Sprintf("[%s] Cleaning temporary resources..", canary.Name))
	canary.Code.DeleteArchive()
}
