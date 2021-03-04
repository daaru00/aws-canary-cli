package iam

import (
	"github.com/aws/aws-sdk-go/service/iam"
)

type clients struct {
	iam *iam.IAM
}
