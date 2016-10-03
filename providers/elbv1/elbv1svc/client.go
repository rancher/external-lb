package elbv1svc

import (
	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elb"
)

const SDKMaxRetries = 3

// ELBClassicService is an abstraction over the AWS SDK that provides methods
// required to manage ELB Classic Load Balancers in a specific region and VPC.
type ELBClassicService struct {
	elbc     *elb.ELB
	ec2c     *ec2.EC2
	metadata *ec2metadata.EC2Metadata
	region   string
	vpcID    string
}

// NewService initializes and returns a new ELBClassicService instance for the specified
// region and VPC using either the specified static credentials or the Instance IAM role.
func NewService(accessKey, secretKey, region, vpcID string) (*ELBClassicService, error) {
	logrus.Debugf("NewService => accessKey: ***, secretKey: ***, region %s, vpcID %s",
		region, vpcID)

	var creds *credentials.Credentials
	if accessKey != "" && secretKey != "" {
		// static credentials
		creds = credentials.NewStaticCredentials(accessKey, secretKey, "")
	} else {
		// IAM instance role
		creds = credentials.NewCredentials(&ec2rolecreds.EC2RoleProvider{
			Client: ec2metadata.New(session.New()),
		})
	}

	awsConfig := aws.NewConfig().
		WithLogger(aws.NewDefaultLogger()).
		WithRegion(region).
		WithMaxRetries(SDKMaxRetries).
		WithCredentials(creds)

	sess, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, err
	}

	service := &ELBClassicService{
		elbc:     elb.New(sess),
		ec2c:     ec2.New(sess),
		metadata: ec2metadata.New(sess),
		region:   region,
		vpcID:    vpcID,
	}

	return service, nil
}

// CheckAPIConnection checks the connection to the AWS API.
func (svc *ELBClassicService) CheckAPIConnection() error {
	logrus.Debug("CheckAPIConnection")
	_, err := svc.ec2c.DescribeInstances(&ec2.DescribeInstancesInput{
		DryRun: aws.Bool(true),
	})
	if err != nil && IsAWSErr(err, AWSErrDryRunOperation) {
		return nil
	}

	return err
}
