package canary

import (
	"bytes"
	"fmt"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/synthetics"
	"github.com/daaru00/aws-canary-cli/internal/iam"
)

type clients struct {
	synthetics *synthetics.Synthetics
	s3         *s3.S3
	s3uploader *s3manager.Uploader
	lambda     *lambda.Lambda
	sts        *sts.STS
}

// Schedule configuration
type Schedule struct {
	DurationInSeconds int64  `yaml:"duration" json:"duration"`
	Expression        string `yaml:"expression" json:"expression"`
}

// VpcConfig configuration
type VpcConfig struct {
	SecurityGroupIds []string `yaml:"securityGroups" json:"securityGroups"`
	SubnetIDs        []string `yaml:"subnets" json:"subnets"`
}

// RetentionConfig configuration
type RetentionConfig struct {
	FailureRetentionPeriod int64 `yaml:"failure" json:"failure"`
	SuccessRetentionPeriod int64 `yaml:"success" json:"success"`
}

// Canary structure
type Canary struct {
	clients *clients
	region  *string

	Name                 string               `yaml:"name" json:"name"`
	Retention            RetentionConfig      `yaml:"retention" json:"retention"`
	RuntimeVersion       string               `yaml:"runtime" json:"runtime"`
	Tags                 map[string]string    `yaml:"tags" json:"tags"`
	Code                 Code                 `yaml:"code" json:"code"`
	EnvironmentVariables map[string]string    `yaml:"env" json:"env"`
	ActiveTracing        bool                 `yaml:"tracing" json:"tracing"`
	MemoryInMB           int64                `yaml:"memory" json:"memory"`
	TimeoutInSeconds     int64                `yaml:"timeout" json:"timeout"`
	Schedule             Schedule             `yaml:"schedule" json:"schedule"`
	VpcConfig            VpcConfig            `yaml:"vpc" json:"vpc"`
	RoleName             string               `yaml:"role" json:"role"`
	PolicyStatements     []iam.StatementEntry `yaml:"policies" json:"policies"`
}

// New creates a new Canary
func New(ses *session.Session, name string) *Canary {
	clients := &clients{
		synthetics: synthetics.New(ses),
		s3:         s3.New(ses),
		s3uploader: s3manager.NewUploader(ses),
		lambda:     lambda.New(ses),
		sts:        sts.New(ses),
	}

	return &Canary{
		clients: clients,
		region:  ses.Config.Region,

		Name:           name,
		RuntimeVersion: "syn-nodejs-puppeteer-3.1",
		Retention: RetentionConfig{
			FailureRetentionPeriod: 31,
			SuccessRetentionPeriod: 31,
		},
		Code: Code{
			clients: clients,

			Handler: "index.handler",
			Src:     "./",
		},
		ActiveTracing:        false,
		TimeoutInSeconds:     840, // 14 minutes
		MemoryInMB:           1000,
		EnvironmentVariables: nil,
		Schedule: Schedule{
			DurationInSeconds: 0,
			Expression:        "rate(0 hour)",
		},
	}
}

// GetFlatTags return tags as flat string
func (c *Canary) GetFlatTags(separator string) *string {
	flat := ""

	// Iterate over tags and concatenated them
	for _, value := range c.Tags {
		if len(value) == 0 {
			continue
		}
		if len(flat) != 0 {
			flat += separator
		}
		flat += fmt.Sprintf("%s", value)
	}

	return &flat
}

// IsDeployed check if Canary name is present in current AWS account
func (c *Canary) IsDeployed() bool {
	_, err := c.clients.synthetics.GetCanary(&synthetics.GetCanaryInput{
		Name: &c.Name,
	})
	return err == nil
}

// Deploy canary
func (c *Canary) Deploy(role *iam.Role, artifactBucketLocation *string) error {
	var err error

	// Elaborate code config
	var codeInputConfig *synthetics.CanaryCodeInput
	if len(c.Code.archives3bucket) != 0 && len(c.Code.archives3key) != 0 {
		// Set S3 path for code
		codeInputConfig = &synthetics.CanaryCodeInput{
			Handler:  &c.Code.Handler,
			S3Bucket: &c.Code.archives3bucket,
			S3Key:    &c.Code.archives3key,
		}
	} else {
		// Load archive path
		data, err := c.Code.ReadArchive()
		if err != nil {
			return err
		}

		// Set zip file code
		codeInputConfig = &synthetics.CanaryCodeInput{
			Handler: &c.Code.Handler,
			ZipFile: data,
		}
	}

	// Elaborate run config
	runConfig := &synthetics.CanaryRunConfigInput{
		ActiveTracing:        &c.ActiveTracing,
		EnvironmentVariables: aws.StringMap(c.EnvironmentVariables),
		MemoryInMB:           &c.MemoryInMB,
		TimeoutInSeconds:     &c.TimeoutInSeconds,
	}

	// Elaborate schedule config
	scheduleConfig := &synthetics.CanaryScheduleInput{
		DurationInSeconds: &c.Schedule.DurationInSeconds,
		Expression:        &c.Schedule.Expression,
	}

	// Elaborate VPC configs
	var vpcConfig *synthetics.VpcConfigInput
	if c.HasVpcConfig() {
		vpcConfig = &synthetics.VpcConfigInput{
			SecurityGroupIds: aws.StringSlice(c.VpcConfig.SecurityGroupIds),
			SubnetIds:        aws.StringSlice(c.VpcConfig.SubnetIDs),
		}
	}

	// Check if Canary is already deployed
	if c.IsDeployed() == false {
		input := &synthetics.CreateCanaryInput{
			Name:                         &c.Name,
			ArtifactS3Location:           artifactBucketLocation,
			ExecutionRoleArn:             role.Arn,
			FailureRetentionPeriodInDays: &c.Retention.FailureRetentionPeriod,
			SuccessRetentionPeriodInDays: &c.Retention.SuccessRetentionPeriod,
			RunConfig:                    runConfig,
			RuntimeVersion:               &c.RuntimeVersion,
			Schedule:                     scheduleConfig,
			Code:                         codeInputConfig,
			Tags:                         aws.StringMap(c.Tags),
		}

		// Setup VPc config only if set
		if vpcConfig != nil {
			input.SetVpcConfig(vpcConfig)
		}

		// Create canary
		_, err = c.clients.synthetics.CreateCanary(input)
	} else {
		input := &synthetics.UpdateCanaryInput{
			Name:                         &c.Name,
			ExecutionRoleArn:             role.Arn,
			FailureRetentionPeriodInDays: &c.Retention.FailureRetentionPeriod,
			SuccessRetentionPeriodInDays: &c.Retention.SuccessRetentionPeriod,
			RunConfig:                    runConfig,
			RuntimeVersion:               &c.RuntimeVersion,
			Schedule:                     scheduleConfig,
			Code:                         codeInputConfig,
		}

		// Setup VPc config only if set
		if vpcConfig != nil {
			input.SetVpcConfig(vpcConfig)
		}

		// Update canary
		_, err = c.clients.synthetics.UpdateCanary(input)
	}

	return err
}

// UpdateTags update canary tags
func (c *Canary) UpdateTags(region *string, account *string) error {
	// Build ARN
	arn := fmt.Sprintf("arn:aws:synthetics:%s:%s:canary:%s", *region, *account, c.Name)

	// Get current tags
	resTags, err := c.clients.synthetics.ListTagsForResource(&synthetics.ListTagsForResourceInput{
		ResourceArn: &arn,
	})
	if err != nil {
		return err
	}

	// Skip if not tags are set
	if len(c.Tags) == 0 && len(resTags.Tags) == 0 {
		return nil
	}

	// Check tags to add
	tagsToAdd := map[string]string{}
	for key, value := range c.Tags {
		var foundTagKey *string
		for currentTagKey, currentTagValue := range resTags.Tags {
			if key == currentTagKey && value == *currentTagValue {
				foundTagKey = &key
				break
			}
		}

		if foundTagKey == nil {
			tagsToAdd[key] = value
		}
	}

	// Add missing tags
	if len(tagsToAdd) > 0 {
		_, err := c.clients.synthetics.TagResource(&synthetics.TagResourceInput{
			ResourceArn: &arn,
			Tags:        aws.StringMap(tagsToAdd),
		})
		if err != nil {
			return err
		}
	}

	// Check accounts ids to remove
	tagsKeysToRemove := []string{}
	for currentTagKey := range resTags.Tags {
		var foundTagKey *string
		for key := range c.Tags {
			if key == currentTagKey {
				foundTagKey = &key
				break
			}
		}

		if foundTagKey == nil {
			tagsKeysToRemove = append(tagsKeysToRemove, currentTagKey)
		}
	}

	// Remove unused tags
	if len(tagsKeysToRemove) > 0 {
		_, err = c.clients.synthetics.UntagResource(&synthetics.UntagResourceInput{
			ResourceArn: &arn,
			TagKeys:     aws.StringSlice(tagsKeysToRemove),
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// Start canary
func (c *Canary) Start() error {
	_, err := c.clients.synthetics.StartCanary(&synthetics.StartCanaryInput{
		Name: &c.Name,
	})
	return err
}

// Stop canary
func (c *Canary) Stop() error {
	_, err := c.clients.synthetics.StopCanary(&synthetics.StopCanaryInput{
		Name: &c.Name,
	})
	return err
}

// GetStatus return canary status
func (c *Canary) GetStatus() (*synthetics.CanaryStatus, error) {
	res, err := c.clients.synthetics.GetCanary(&synthetics.GetCanaryInput{
		Name: &c.Name,
	})

	return res.Canary.Status, err
}

// GetRuns return canary runs
func (c *Canary) GetRuns() ([]*synthetics.CanaryRun, error) {
	res, err := c.clients.synthetics.GetCanaryRuns(&synthetics.GetCanaryRunsInput{
		Name: &c.Name,
	})

	return res.CanaryRuns, err
}

// GetLastRun return the latest canary run
func (c *Canary) GetLastRun() (*synthetics.CanaryRun, error) {
	runs, err := c.GetRuns()
	if len(runs) > 0 {
		return runs[0], nil
	}
	return nil, err
}

// GetRunLogs return canary run log
func (c *Canary) GetRunLogs(run *synthetics.CanaryRun) (*string, error) {
	log := ""

	// Check if not data are set
	if *run.ArtifactS3Location == "No data" {
		return run.ArtifactS3Location, nil
	}

	// Elaborate bucket name
	artifactPath := *run.ArtifactS3Location
	bucketName := strings.Split(artifactPath, "/")[0]

	// List artifact objects
	listRes, err := c.clients.s3.ListObjects(&s3.ListObjectsInput{
		Bucket: &bucketName,
		Prefix: aws.String(artifactPath[len(bucketName)+1:]),
	})

	if err != nil {
		return &log, err
	}

	// Search for logs
	logKey := ""
	for _, object := range listRes.Contents {
		if path.Ext(*object.Key) == ".txt" {
			logKey = *object.Key
			break
		}
	}

	// Check if log was found
	if len(logKey) == 0 {
		return &log, fmt.Errorf("Cannot find log txt file in artifact bucket s3://%s", artifactPath)
	}

	// Retrieve log file content
	getRes, err := c.clients.s3.GetObject(&s3.GetObjectInput{
		Bucket: &bucketName,
		Key:    &logKey,
	})
	if err != nil {
		return &log, err
	}

	// Read object content
	buf := new(bytes.Buffer)
	buf.ReadFrom(getRes.Body)
	log = buf.String()

	return &log, nil
}

// Remove canary
func (c *Canary) Remove() error {
	// Get canary
	canaryGet, err := c.clients.synthetics.GetCanary(&synthetics.GetCanaryInput{
		Name: &c.Name,
	})
	if err != nil {
		return err
	}

	// Delete canary
	_, err = c.clients.synthetics.DeleteCanary(&synthetics.DeleteCanaryInput{
		Name: &c.Name,
	})
	if err != nil {
		return err
	}

	// Delete related layer (ignore error if not exist)
	layerName := fmt.Sprintf("cwsyn-%s-%s", c.Name, *canaryGet.Canary.Id)
	layerList, _ := c.clients.lambda.ListLayerVersions(&lambda.ListLayerVersionsInput{
		LayerName: &layerName,
	})

	// Delete all layer's versions
	for _, version := range layerList.LayerVersions {
		_, err = c.clients.lambda.DeleteLayerVersion(&lambda.DeleteLayerVersionInput{
			LayerName:     &layerName,
			VersionNumber: version.Version,
		})
		if err != nil {
			return err
		}
	}

	// Delete related function (ignore error if not exist)
	c.clients.lambda.DeleteFunction(&lambda.DeleteFunctionInput{
		FunctionName: &layerName,
	})

	return nil
}

// IsNodeRuntime check if is node runtime
func (c *Canary) IsNodeRuntime() bool {
	return strings.Contains(c.RuntimeVersion, "nodejs")
}

// IsPythonRuntime check if is python runtime
func (c *Canary) IsPythonRuntime() bool {
	return strings.Contains(c.RuntimeVersion, "python")
}

// HasVpcConfig check if is VPC access is configured
func (c *Canary) HasVpcConfig() bool {
	return len(c.VpcConfig.SubnetIDs) > 0
}
