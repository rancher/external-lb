package elbv1svc

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// EC2Instance represents an EC2 instance on AWS
type EC2Instance struct {
	// The ID of the instance.
	ID string
	// The private IP address assigned to the instance.
	PrivateIPAddress string
	// The public IP address assigned to the instance
	PublicIPAddress string
	// The ID of the subnet the instance is running in.
	SubnetID string
	// The IDs of one or more security group assigned to the instance.
	SecurityGroups []string
	// The ID of the VPC the instance is running in.
	VpcID string
}

// GetInstancesByID returns the EC2 instances with the specified IDs.
func (svc *ELBClassicService) GetInstancesByID(ids []string) ([]*EC2Instance, error) {
	filters := []*ec2.Filter{
		NewEC2Filter("instance-id", ids...),
	}
	return svc.LookupInstancesByFilter(filters)
}

// LookupInstancesByIPAddress looks up the EC2 instances with the specified
// IP addresses. The privateIP parameter specifies whether the given IPs
// are public or private IP addresses.
func (svc *ELBClassicService) LookupInstancesByIPAddress(ipAddresses []string, privateIP bool) ([]*EC2Instance, error) {
	filters := []*ec2.Filter{
		NewEC2Filter("vpc-id", svc.vpcID),
	}

	if privateIP {
		filters = append(filters, NewEC2Filter("private-ip-address", ipAddresses...))
	} else {
		filters = append(filters, NewEC2Filter("ip-address", ipAddresses...))
	}

	instances, err := svc.LookupInstancesByFilter(filters)
	if err != nil {
		return nil, fmt.Errorf("Failed to look up instances by IP address: %v", err)
	}

	return instances, nil
}

// LookupInstancesByFilter looks up EC2 instances using the specified filters.
// It returns nil if no matching instances were found.
func (svc *ELBClassicService) LookupInstancesByFilter(filters []*ec2.Filter) ([]*EC2Instance, error) {
	var instances []*EC2Instance
	params := &ec2.DescribeInstancesInput{
		Filters: filters,
	}

	resp, err := svc.ec2c.DescribeInstances(params)
	if err != nil {
		return instances, fmt.Errorf("DescribeInstances SDK error: %v", err)
	}

	if resp == nil || len(resp.Reservations) == 0 {
		return instances, nil
	}

	for _, r := range resp.Reservations {
		for _, ec2instance := range r.Instances {
			securityGroups := make([]string, len(ec2instance.SecurityGroups))
			for i, sg := range ec2instance.SecurityGroups {
				securityGroups[i] = *sg.GroupId
			}
			instance := &EC2Instance{
				ID:               *ec2instance.InstanceId,
				PrivateIPAddress: *ec2instance.PrivateIpAddress,
				PublicIPAddress:  *ec2instance.PublicIpAddress,
				SubnetID:         *ec2instance.SubnetId,
				SecurityGroups:   securityGroups,
			}
			if ec2instance.VpcId != nil {
				instance.VpcID = *ec2instance.VpcId
			}

			instances = append(instances, instance)
		}
	}
	return instances, nil
}

// DescribeSubnets returns the ec2.Subnet structs for the specified subnet IDs.
func (svc *ELBClassicService) DescribeSubnets(ids []string) ([]*ec2.Subnet, error) {
	awsSubnets := make([]*string, len(ids))
	for i, id := range ids {
		awsSubnets[i] = aws.String(id)
	}
	params := &ec2.DescribeSubnetsInput{
		SubnetIds: awsSubnets,
	}
	resp, err := svc.ec2c.DescribeSubnets(params)
	if err != nil {
		return nil, fmt.Errorf("DescribeSubnets SDK error: %v", err)
	}

	return resp.Subnets, nil
}

// AzSubnets returns a map of all availability zones in the service's
// VPC as keys and the ID of one active subnet in that zone as value.
func (svc *ELBClassicService) GetAzSubnets() (map[string]string, error) {
	params := &ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []*string{aws.String(svc.vpcID)},
			},
			{
				Name:   aws.String("state"),
				Values: []*string{aws.String("available")},
			},
		},
	}
	resp, err := svc.ec2c.DescribeSubnets(params)
	if err != nil {
		return nil, fmt.Errorf("DescribeSubnets SDK error: %v", err)
	}

	ret := make(map[string]string)
	for _, s := range resp.Subnets {
		if _, ok := ret[*s.AvailabilityZone]; !ok {
			ret[*s.AvailabilityZone] = *s.SubnetId
		}
	}
	return ret, nil
}

// IsDefaultVPC returns true if the specified VPC is the default
// VPC in the service's region.
func (svc *ELBClassicService) IsDefaultVPC(vpcID string) (bool, error) {
	params := &ec2.DescribeVpcsInput{
		VpcIds: []*string{aws.String(vpcID)},
	}
	resp, err := svc.ec2c.DescribeVpcs(params)
	if err != nil {
		return false, fmt.Errorf("DescribeVpcs SDK error: %v", err)
	}
	if len(resp.Vpcs) == 0 {
		return false, fmt.Errorf("Could not find VPC with ID: %s", vpcID)
	}
	return *resp.Vpcs[0].IsDefault, nil
}
