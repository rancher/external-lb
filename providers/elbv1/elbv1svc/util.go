package elbv1svc

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elb"
)

const (
	AWSErrPermissionNotFound     = "InvalidPermission.NotFound"
	AWSErrPermissionDuplicate    = "InvalidPermission.Duplicate"
	AWSErrSecurityGroupNotFound  = "InvalidGroup.NotFound"
	AwsErrSecurityGroupDuplicate = "InvalidGroup.Duplicate"
	AWSErrTargetGroupNotFound    = "TargetGroupNotFound"
	AWSErrRuleNotFound           = "RuleNotFound"
	AWSErrDependendyViolation    = "DependencyViolation"
	AWSErrDryRunOperation        = "DryRunOperation"
	AWSErrLoadBalancerNotFound   = "LoadBalancerNotFound"
	AWSErrResourceInUse          = "ResourceInUse"
)

func NewEC2Filter(name string, values ...string) *ec2.Filter {
	awsValues := make([]*string, len(values))
	for i, val := range values {
		awsValues[i] = aws.String(val)
	}
	return &ec2.Filter{
		Name:   aws.String(name),
		Values: awsValues,
	}
}

func IsAWSErr(err error, code string) bool {
	switch e := err.(type) {
	case awserr.Error:
		if e.Code() == code {
			return true
		}
	}
	return false
}

// takes a map[string]string and converts it to an elb.Tag slice.
func elbTags(tags map[string]string) []*elb.Tag {
	s := make([]*elb.Tag, len(tags))
	i := 0
	for k, v := range tags {
		s[i] = &elb.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		}
		i++
	}
	return s
}

// takes an elb.Tag slice and converts it to a map[string]string.
func mapTags(tags []*elb.Tag) map[string]string {
	m := make(map[string]string, len(tags))
	for _, t := range tags {
		m[*t.Key] = *t.Value
	}
	return m
}
