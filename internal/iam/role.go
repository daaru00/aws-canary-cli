package iam

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	awsinternal "github.com/daaru00/aws-canary-cli/internal/aws"
)

// Role structure
type Role struct {
	clients *clients

	Name         *string
	Arn          *string
	InlinePolicy *Policy
}

// NewRole creates a new IAM Role for Canary
func NewRole(ses *session.Session, nameOrArn *string) *Role {
	var arn string
	var name string

	// Get account id
	accountID := awsinternal.GetCallerAccountID(ses)

	// Check if an arn is provided
	if strings.HasPrefix(*nameOrArn, "arn:") == true {
		arnParts := strings.Split(name, "/")
		name = arnParts[1]
		arn = *nameOrArn
	} else {
		name = *nameOrArn

		arn = fmt.Sprintf("arn:aws:iam::%s:role/%s", *accountID, name)
	}

	return &Role{
		clients: &clients{
			iam.New(ses),
		},
		Name:         &name,
		Arn:          &arn,
		InlinePolicy: nil,
	}
}

// IsDeployed check if IAM Role name is present in current AWS account
func (r *Role) IsDeployed() bool {
	_, err := r.clients.iam.GetRole(&iam.GetRoleInput{
		RoleName: r.Name,
	})
	return err == nil
}

// SetInlinePolicy set an inline policy
func (r *Role) SetInlinePolicy(policy *Policy) error {
	r.InlinePolicy = policy
	return nil
}

// Deploy IAM role
func (r *Role) Deploy() error {
	var needWait bool

	// Check if not deployed
	if r.IsDeployed() == false {
		// Create role
		_, err := r.clients.iam.CreateRole(&iam.CreateRoleInput{
			RoleName: r.Name,
			AssumeRolePolicyDocument: aws.String(`{
				"Version": "2012-10-17",
				"Statement": [{
					"Effect": "Allow",
					"Principal": {
						"Service": [
							"lambda.amazonaws.com"
						]
					},
					"Action": [
						"sts:AssumeRole"
					]
				}]
			}`),
		})
		if err != nil {
			return err
		}

		// Wait until policy is fully created
		err = r.clients.iam.WaitUntilRoleExists(&iam.GetRoleInput{
			RoleName: r.Name,
		})
		if err != nil {
			return err
		}

		// Set a dummy sleep to avoid IAM role not recognized as valid Lambda role
		needWait = true
	}

	// Check for inline policy
	if r.InlinePolicy != nil {
		// Render policy
		policyDoc, err := r.InlinePolicy.Render()
		if err != nil {
			return err
		}

		// Add role policy
		_, err = r.clients.iam.PutRolePolicy(&iam.PutRolePolicyInput{
			RoleName:       r.Name,
			PolicyName:     r.InlinePolicy.Name,
			PolicyDocument: policyDoc,
		})
		if err != nil {
			return err
		}
	}

	// Do a dummy sleep to avoid IAM role not recognized as valida Lambda role.
	if needWait {
		time.Sleep(10 * 1000 * time.Millisecond) // Yes, 10 is a magic number.
	}

	return nil
}

// Remove IAM role
func (r *Role) Remove() error {
	// Get role inline policy
	_, err := r.clients.iam.GetRolePolicy(&iam.GetRolePolicyInput{
		RoleName:   r.Name,
		PolicyName: r.InlinePolicy.Name,
	})
	if err != nil {
		return err
	}

	// Delete inline policy
	_, err = r.clients.iam.DeleteRolePolicy(&iam.DeleteRolePolicyInput{
		RoleName:   r.Name,
		PolicyName: r.InlinePolicy.Name,
	})
	if err != nil {
		return err
	}

	// Delete role
	_, err = r.clients.iam.DeleteRole(&iam.DeleteRoleInput{
		RoleName: r.Name,
	})
	if err != nil {
		return err
	}

	return nil
}
