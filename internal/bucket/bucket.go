package bucket

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type clients struct {
	s3 *s3.S3
}

// Bucket structure
type Bucket struct {
	clients *clients

	Name     *string
	Location *string
}

// New creates a bucket
func New(ses *session.Session, name *string) *Bucket {
	location := fmt.Sprintf("s3://%s", *name)

	return &Bucket{
		clients: &clients{
			s3.New(ses),
		},
		Name:     name,
		Location: &location,
	}
}

// IsDeployed check if IAM Bucket name is present in current AWS account
func (b *Bucket) IsDeployed() bool {
	_, err := b.clients.s3.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: b.Name,
	})
	return err == nil
}

// Deploy Bucket
func (b *Bucket) Deploy() error {

	// Check if bucket is already deployed
	if b.IsDeployed() == false {
		// Create new Bucket
		_, err := b.clients.s3.CreateBucket(&s3.CreateBucketInput{
			Bucket: b.Name,
		})
		if err != nil {
			return err
		}
	}

	// Put public ACL lock
	_, err := b.clients.s3.PutPublicAccessBlock(&s3.PutPublicAccessBlockInput{
		Bucket: b.Name,
		PublicAccessBlockConfiguration: &s3.PublicAccessBlockConfiguration{
			BlockPublicAcls:       aws.Bool(true),
			BlockPublicPolicy:     aws.Bool(true),
			IgnorePublicAcls:      aws.Bool(true),
			RestrictPublicBuckets: aws.Bool(true),
		},
	})

	return err
}

// DeployLifecycleConfigurationExpires deploy lifecycle configuration for expiration
func (b *Bucket) DeployLifecycleConfigurationExpires(days int64) error {
	_, err := b.clients.s3.PutBucketLifecycleConfiguration(&s3.PutBucketLifecycleConfigurationInput{
		Bucket: b.Name,
		LifecycleConfiguration: &s3.BucketLifecycleConfiguration{
			Rules: []*s3.LifecycleRule{
				{
					Prefix: aws.String(""),
					Status: aws.String("Enabled"),
					Expiration: &s3.LifecycleExpiration{
						Days: aws.Int64(days),
					},
				},
			},
		},
	})
	return err
}

// Empty Bucket
func (b *Bucket) Empty() error {

	// Check if bucket is not deployed
	if b.IsDeployed() == false {
		return nil
	}

	var listRes *s3.ListObjectsV2Output
	for {
		// List objects
		listRes, err := b.clients.s3.ListObjectsV2(&s3.ListObjectsV2Input{
			Bucket:            b.Name,
			ContinuationToken: listRes.ContinuationToken,
		})
		if err != nil {
			return err
		}

		// Collect keys
		keysToDelete := []*s3.ObjectIdentifier{}
		for _, object := range listRes.Contents {
			keysToDelete = append(keysToDelete, &s3.ObjectIdentifier{
				Key: object.Key,
			})
		}

		// Delete objects
		b.clients.s3.DeleteObjects(&s3.DeleteObjectsInput{
			Bucket: b.Name,
			Delete: &s3.Delete{
				Objects: keysToDelete,
			},
		})

		// Check if list is compleated
		if *listRes.IsTruncated {
			break
		}
	}

	return nil
}

// Remove Bucket
func (b *Bucket) Remove() error {

	// Check if bucket is not deployed
	if b.IsDeployed() == false {
		return nil
	}

	// Delete Bucket
	_, err := b.clients.s3.DeleteBucket(&s3.DeleteBucketInput{
		Bucket: b.Name,
	})

	return err
}
