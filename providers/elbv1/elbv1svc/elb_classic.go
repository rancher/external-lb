package elbv1svc

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elb"
)

// ELB backend instance states
const (
	InService    = "InService"
	OutOfService = "OutOfService"
	Unknown      = "Unknown"
)

// GetLoadBalancers returns the LoadBalancerDescription struct
// for the specified load balancer or nil if it was not found.
func (svc *ELBClassicService) GetLoadBalancerByName(name string) (*elb.LoadBalancerDescription, error) {
	logrus.Debugf("GetLoadBalancerByName => %s", name)
	desc, err := svc.GetLoadBalancers(name)
	if err != nil {
		if IsAWSErr(err, AWSErrLoadBalancerNotFound) {
			return nil, nil
		}
		return nil, err
	}

	if len(desc) > 0 {
		return desc[0], nil
	}

	return nil, nil
}

// GetLoadBalancers returns the LoadBalancerDescription structs
// for the specified load balancer. Otherwise returns all.
func (svc *ELBClassicService) GetLoadBalancers(names ...string) ([]*elb.LoadBalancerDescription, error) {
	logrus.Debugf("GetLoadBalancers => %v", names)
	params := &elb.DescribeLoadBalancersInput{}
	if len(names) > 0 {
		awsNames := make([]*string, len(names))
		for i, n := range names {
			awsNames[i] = aws.String(n)
		}
		params.LoadBalancerNames = awsNames
	}

	resp, err := svc.elbc.DescribeLoadBalancers(params)
	if err != nil {
		return nil, err
	}

	return resp.LoadBalancerDescriptions, nil
}

// DescribeLBTags returns the tags for the specified load balancers
// as map of load balancer name => map[string]string.
func (svc *ELBClassicService) DescribeLBTags(loadBalancerNames []string) (map[string]map[string]string, error) {
	logrus.Debugf("DescribeLBTags => %v", loadBalancerNames)
	awsNames := make([]*string, len(loadBalancerNames))
	for i, n := range loadBalancerNames {
		awsNames[i] = aws.String(n)
	}

	params := &elb.DescribeTagsInput{
		LoadBalancerNames: awsNames,
	}
	resp, err := svc.elbc.DescribeTags(params)
	if err != nil {
		return nil, fmt.Errorf("DescribeTags SDK error: %v", err)
	}

	ret := make(map[string]map[string]string)
	for _, desc := range resp.TagDescriptions {
		tags := mapTags(desc.Tags)
		ret[*desc.LoadBalancerName] = tags
	}

	return ret, nil
}

// AddLBTags adds the specified tags to the specified load balancer.
func (svc *ELBClassicService) AddLBTags(loadBalancerName string, tags map[string]string) error {
	logrus.Debugf("AddLBTags => name: %s, tags %v", loadBalancerName, tags)
	awsTags := elbTags(tags)
	params := &elb.AddTagsInput{
		LoadBalancerNames: []*string{
			aws.String(loadBalancerName),
		},
		Tags: awsTags,
	}

	_, err := svc.elbc.AddTags(params)
	if err != nil {
		return fmt.Errorf("AddTags SDK error: %v", err)
	}

	return nil
}

// RemoveLBTags adds the specified tags to the specified load balancer.
func (svc *ELBClassicService) RemoveLBTag(loadBalancerName string, tagKey string) error {
	logrus.Debugf("RemoveLBTag => name: %s, tagKey %s", loadBalancerName, tagKey)
	params := &elb.RemoveTagsInput{
		LoadBalancerNames: []*string{
			aws.String(loadBalancerName),
		},
		Tags: []*elb.TagKeyOnly{
			{
				Key: aws.String(tagKey),
			},
		},
	}
	_, err := svc.elbc.RemoveTags(params)
	if err != nil {
		return fmt.Errorf("RemoveTags SDK error: %v", err)
	}

	return nil
}

// EnsureListenerInstancePort ensures that the specified listener
// is configured to forward traffic to the specified instance port.
// If not, it recreates the listener and configures it so, while
// maintaining all it's other properties.
func (svc *ELBClassicService) EnsureListenerInstancePort(
	loadBalancerName string, instancePort int64, listenerDesc *elb.ListenerDescription) error {
	logrus.Debugf("EnsureListenerInstancePort => name: %s, port: %d, listener: %v",
		loadBalancerName, instancePort, listenerDesc)

	loadBalancerPort := *listenerDesc.Listener.LoadBalancerPort
	if *listenerDesc.Listener.InstancePort == instancePort {
		logrus.Debugf("Listener %s:%d is already using instance port %d",
			loadBalancerName, loadBalancerPort, instancePort)
		return nil
	}

	logrus.Debugf("Recreating listener %s:%d with instance port %d",
		loadBalancerName, loadBalancerPort, instancePort)

	{
		params := &elb.DeleteLoadBalancerListenersInput{
			LoadBalancerName: aws.String(loadBalancerName),
			LoadBalancerPorts: []*int64{
				aws.Int64(loadBalancerPort),
			},
		}

		_, err := svc.elbc.DeleteLoadBalancerListeners(params)
		if err != nil {
			return fmt.Errorf("DeleteLoadBalancerListeners SDK error: %v", err)
		}
	}

	{
		params := &elb.CreateLoadBalancerListenersInput{
			Listeners: []*elb.Listener{
				{
					InstancePort:     aws.Int64(instancePort),
					LoadBalancerPort: aws.Int64(loadBalancerPort),
					Protocol:         listenerDesc.Listener.Protocol,
					InstanceProtocol: listenerDesc.Listener.InstanceProtocol,
					SSLCertificateId: listenerDesc.Listener.SSLCertificateId,
				},
			},
			LoadBalancerName: aws.String(loadBalancerName),
		}

		_, err := svc.elbc.CreateLoadBalancerListeners(params)
		if err != nil {
			return fmt.Errorf("CreateLoadBalancerListeners SDK error: %v", err)
		}
	}

	logrus.Debugf("Restoring policies for Listener %s/%d: %v",
		loadBalancerName, loadBalancerPort, listenerDesc.PolicyNames)

	// Restoring policies
	if len(listenerDesc.PolicyNames) > 0 {
		params := &elb.SetLoadBalancerPoliciesOfListenerInput{
			LoadBalancerName: aws.String(loadBalancerName),
			LoadBalancerPort: aws.Int64(loadBalancerPort),
			PolicyNames:      listenerDesc.PolicyNames,
		}

		_, err := svc.elbc.SetLoadBalancerPoliciesOfListener(params)
		if err != nil {
			return fmt.Errorf("SetLoadBalancerPoliciesOfListener SDK error: %v", err)
		}
	}

	return nil
}

// EnsureHealthCheckPort ensures that the specified healthcheck
// uses the specified instance port. If not, it configures it
// so, while maintaining all it's other properties.
func (svc *ELBClassicService) EnsureHealthCheckPort(
	loadBalancerName string, instancePort int64, healthcheck *elb.HealthCheck) error {
	logrus.Debugf("EnsureHealthCheckPort => name: %s, port: %d, health: %v",
		loadBalancerName, instancePort, healthcheck)

	target := *healthcheck.Target
	logrus.Debugf("Current health check target: %s", target)
	var proto, port, path string
	// health check target format:
	// <PROTO>:<PORT>[/<PATH>]
	parts := strings.Split(target, ":")
	proto = parts[0]
	parts_ := strings.SplitN(parts[1], "/", 2)
	if len(parts_) > 1 {
		port = parts_[0]
		path = parts_[1]
	} else {
		port = parts[1]
	}

	portInt64, err := strconv.ParseInt(port, 10, 32)
	if err != nil {
		return fmt.Errorf("Failed to get integer of health check targer port: %v", err)
	}

	logrus.Debugf("Current health check port: %d", portInt64)

	if portInt64 == instancePort {
		logrus.Debugf("Health check already uses port %d", instancePort)
		return nil
	}

	newTarget := fmt.Sprintf("%s:%s", proto, port)
	if len(path) > 0 {
		newTarget = fmt.Sprintf("%s/%s", newTarget, path)
	}

	logrus.Debugf("New health check target: %s", newTarget)

	params := &elb.ConfigureHealthCheckInput{
		HealthCheck: &elb.HealthCheck{
			HealthyThreshold:   healthcheck.HealthyThreshold,
			Interval:           healthcheck.Interval,
			Target:             aws.String(newTarget),
			Timeout:            healthcheck.Timeout,
			UnhealthyThreshold: healthcheck.UnhealthyThreshold,
		},
		LoadBalancerName: aws.String(loadBalancerName),
	}
	_, err = svc.elbc.ConfigureHealthCheck(params)
	if err != nil {
		return fmt.Errorf("ConfigureHealthCheck SDK error: %v", err)
	}

	return nil
}

// RegisterInstances registers the specified EC2
// instances with the specified load balancer.
func (svc *ELBClassicService) RegisterInstances(loadBalancerName string, instanceIds []string) error {
	logrus.Debugf("RegisterInstances => name: %s instances: %v",
		loadBalancerName, instanceIds)

	awsInstances := make([]*elb.Instance, len(instanceIds))
	for i, id := range instanceIds {
		awsInstances[i] = &elb.Instance{
			InstanceId: aws.String(id),
		}
	}

	params := &elb.RegisterInstancesWithLoadBalancerInput{
		Instances:        awsInstances,
		LoadBalancerName: aws.String(loadBalancerName),
	}
	_, err := svc.elbc.RegisterInstancesWithLoadBalancer(params)
	if err != nil {
		return fmt.Errorf("RegisterInstancesWithLoadBalancer SDK error: %v", err)
	}

	return nil
}

// DergisterInstances deregisters the specified EC2
// instances from the specified load balancer.
func (svc *ELBClassicService) DeregisterInstances(loadBalancerName string, instanceIds []string) error {
	logrus.Debugf("DeregisterInstances => name: %s instances: %v",
		loadBalancerName, instanceIds)

	awsInstances := make([]*elb.Instance, len(instanceIds))
	for i, id := range instanceIds {
		awsInstances[i] = &elb.Instance{
			InstanceId: aws.String(id),
		}
	}

	params := &elb.DeregisterInstancesFromLoadBalancerInput{
		Instances:        awsInstances,
		LoadBalancerName: aws.String(loadBalancerName),
	}
	_, err := svc.elbc.DeregisterInstancesFromLoadBalancer(params)
	if err != nil {
		return fmt.Errorf("DeregisterInstancesFromLoadBalancer SDK error: %v", err)
	}

	return nil
}

// DescribeInstanceHealth returns the state of the specified or all instances
// that are currently registered with the specified load balancer.
func (svc *ELBClassicService) DescribeInstanceHealth(loadBalancerName string, instanceIds ...string) ([]*elb.InstanceState, error) {
	params := &elb.DescribeInstanceHealthInput{
		LoadBalancerName: aws.String(loadBalancerName),
	}

	if len(instanceIds) > 0 {
		awsInstances := make([]*elb.Instance, len(instanceIds))
		for i, id := range instanceIds {
			awsInstances[i] = &elb.Instance{
				InstanceId: aws.String(id),
			}
		}
		params.Instances = awsInstances
	}

	resp, err := svc.elbc.DescribeInstanceHealth(params)
	if err != nil {
		return nil, err
	}

	logrus.Debugf("DescribeInstanceHealth result: %v", resp.InstanceStates)
	return resp.InstanceStates, nil
}

// GetRegisteredInstances returns the IDs of all instances that are currently
// registered to the load balancer. Includes instances whose registration is
// currently in progress and excludes those whose deregistration is currently
// in progress.
func (svc *ELBClassicService) GetRegisteredInstances(loadBalancerName string) ([]string, error) {
	instanceStates, err := svc.DescribeInstanceHealth(loadBalancerName)
	if err != nil {
		return nil, err
	}

	var instanceIds []string
	for _, state := range instanceStates {
		switch *state.State {
		case InService:
			instanceIds = append(instanceIds, *state.InstanceId)
		case OutOfService, Unknown:
			if strings.Contains(*state.Description, "deregistration") ||
				strings.Contains(*state.Description, "not currently registered") {
				logrus.Debugf("Skipping instance that's not registered or being deregistered: %s", *state.InstanceId)
				continue
			}
			instanceIds = append(instanceIds, *state.InstanceId)
		}
	}

	logrus.Debugf("GetRegisteredInstances result: %v", instanceIds)
	return instanceIds, nil
}
