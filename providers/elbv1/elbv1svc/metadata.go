package elbv1svc

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
)

// GetInstanceInfo returns the VPC ID and region of the EC2 instance
// on which the application is running.
func GetInstanceInfo() (vpcID, region string, errOut error) {
	mClient := ec2metadata.New(session.New())
	if mClient.Available() == false {
		errOut = fmt.Errorf("EC2 Metadata service is unavailable. " +
			"Make sure this application is running on an EC2 Instance.")
		return
	}
	region, err := mClient.Region()
	if err != nil {
		errOut = fmt.Errorf("Could not get region of the instance: %v", err)
		return
	}

	macs, err := mClient.GetMetadata("network/interfaces/macs/")
	if err != nil {
		errOut = fmt.Errorf("Could not get interfaces of the instance: %v", err)
		return
	}

	for _, mac := range strings.Split(macs, "\n") {
		if len(mac) == 0 {
			continue
		}
		path := fmt.Sprintf("network/interfaces/macs/%svpc-id", mac)
		vpcID, err = mClient.GetMetadata(path)
		if err != nil {
			continue
		}
		return
	}

	errOut = fmt.Errorf("Failed to determine the VPC of the instance from metadata")
	return
}
