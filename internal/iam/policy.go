package iam

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	awsinternal "github.com/daaru00/aws-canary-cli/internal/aws"
	"github.com/daaru00/aws-canary-cli/internal/bucket"
)

// Policy structure
type Policy struct {
	clients    *clients
	statements []StatementEntry

	Name *string
	Arn  *string
}

// PolicyDocument structure
type PolicyDocument struct {
	Version   string
	Statement []StatementEntry
}

// StatementEntry structure
type StatementEntry struct {
	Effect    string    `yaml:"Effect" json:"Effect"`
	Action    []string  `yaml:"Action" json:"Action"`
	Resource  []string  `yaml:"Resource" json:"Resource"`
	Condition Condition `yaml:"Condition" json:"Condition"`
}

// Condition structure
type Condition struct {
	StringEquals map[string]string `json:"StringEquals,omitempty"`
}

// NewPolicy creates a new IAM Policy for Canary
func NewPolicy(ses *session.Session, nameOrArn *string) *Policy {
	var arn string
	var name string

	// Get account id
	accountID := awsinternal.GetCallerAccountID(ses)

	// Check if an arn is provided
	if len(*nameOrArn) > 0 {
		if strings.HasPrefix(*nameOrArn, "arn:") == true {
			arn = *nameOrArn
			arnParts := strings.Split(*nameOrArn, "/")
			name = arnParts[len(arnParts)-1]
		} else {
			name = *nameOrArn

			arn = fmt.Sprintf("arn:aws:iam::%s:policy/%s", *accountID, name)
		}
	}

	return &Policy{
		clients: &clients{
			iam: iam.New(ses),
		},
		statements: []StatementEntry{},

		Name: &name,
		Arn:  &arn,
	}
}

// IsDeployed check if IAM Policy name is present in current AWS account
func (p *Policy) IsDeployed() bool {
	_, err := p.clients.iam.GetPolicy(&iam.GetPolicyInput{
		PolicyArn: p.Arn,
	})

	return err == nil
}

// AddStatement add statement to policy
func (p *Policy) AddStatement(statement StatementEntry) {
	p.statements = append(p.statements, statement)
}

// AddArtifactBucketPermission add s3 artifact permissions statements to policy
func (p *Policy) AddArtifactBucketPermission(artifactBucket *bucket.Bucket) {
	p.statements = append([]StatementEntry{
		{
			Effect: "Allow",
			Action: []string{
				"s3:PutObject",
			},
			Resource: []string{
				fmt.Sprintf("arn:aws:s3:::%s/*", *artifactBucket.Name),
			},
		},
		{
			Effect: "Allow",
			Action: []string{
				"s3:GetBucketLocation",
			},
			Resource: []string{
				fmt.Sprintf("arn:aws:s3:::%s", *artifactBucket.Name),
			},
		},
		{
			Effect: "Allow",
			Action: []string{
				"s3:ListAllMyBuckets",
			},
			Resource: []string{
				"*",
			},
		},
	}, p.statements...)
}

// AddXRayPermission add xray permissions statements to policy
func (p *Policy) AddXRayPermission() {
	p.statements = append([]StatementEntry{
		{
			Effect: "Allow",
			Action: []string{
				"xray:PutTraceSegments",
			},
			Resource: []string{
				"*",
			},
		},
	}, p.statements...)
}

// AddLogPermission add cloudwatch log permissions statements to policy
func (p *Policy) AddLogPermission(region *string, accountID *string) {
	p.statements = append([]StatementEntry{
		{
			Effect: "Allow",
			Action: []string{
				"logs:CreateLogStream",
				"logs:PutLogEvents",
				"logs:CreateLogGroup",
			},
			Resource: []string{
				fmt.Sprintf("arn:aws:logs:%s:%s:log-group:/aws/lambda/cwsyn-*", *region, *accountID),
			},
		},
	}, p.statements...)
}

// AddMetricsPermission add cloudwatch metrics permissions statements to policy
func (p *Policy) AddMetricsPermission() {
	p.statements = append([]StatementEntry{
		{
			Effect: "Allow",
			Action: []string{
				"cloudwatch:PutMetricData",
			},
			Resource: []string{
				"*",
			},
			Condition: Condition{
				StringEquals: map[string]string{
					"cloudwatch:namespace": "CloudWatchSynthetics",
				},
			},
		},
	}, p.statements...)
}

// AddSSMParamersPermission add SSM parameters metrics permissions statements to policy
func (p *Policy) AddSSMParamersPermission(region *string, accountID *string) {
	p.statements = append([]StatementEntry{
		{
			Effect: "Allow",
			Action: []string{
				"ssm:GetParameter*",
			},
			Resource: []string{
				fmt.Sprintf("arn:aws:ssm:%s:%s:parameter/cwsyn/*", *region, *accountID),
			},
		},
	}, p.statements...)
}

// AddVPCPermissions add permission for VPC
func (p *Policy) AddVPCPermissions() {
	p.statements = append([]StatementEntry{
		{
			Effect: "Allow",
			Action: []string{
				"ec2:CreateNetworkInterface",
				"ec2:DescribeNetworkInterface",
				"ec2:DescribeNetworkInterfaces",
				"ec2:DeleteNetworkInterface",
			},
			Resource: []string{
				"*",
			},
		},
	}, p.statements...)
}

// Render IAM Policy
func (p *Policy) Render() (*string, error) {
	str := "{}"

	// Build policy document
	policy := PolicyDocument{
		Version:   "2012-10-17",
		Statement: p.statements,
	}

	// Generate policy document
	doc, err := json.Marshal(&policy)
	if err != nil {
		return &str, err
	}

	// Convert to string
	str = string(doc)
	return &str, err
}
