package awselbv1

import (
	"fmt"
	"os"
	"strconv"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/external-lb/model"
	"github.com/rancher/external-lb/providers"
	"github.com/rancher/external-lb/providers/elbv1/elbv1svc"
)

const (
	ProviderName = "AWS ELB Classic"
	ProviderSlug = "elbv1"
)

const (
	TagNameTargetPool  = "external-lb/targetPoolName"
	TagNameServicePort = "external-lb/servicePort"
)

const (
	EnvVarAWSAccessKey = "ELBV1_AWS_ACCESS_KEY"
	EnvVarAWSSecretKey = "ELBV1_AWS_SECRET_KEY"
	EnvVarAWSRegion    = "ELBV1_AWS_REGION"
	EnvVarAWSVpcID     = "ELBV1_AWS_VPCID"
	EnvVarUsePrivateIP = "ELBV1_USE_PRIVATE_IP"
)

// AWSELBv1Provider implements the providers.Provider interface.
type AWSELBv1Provider struct {
	svc          *elbv1svc.ELBClassicService
	region       string
	vpcID        string
	usePrivateIP bool
}

func init() {
	providers.RegisterProvider(ProviderSlug, new(AWSELBv1Provider))
}

func (p *AWSELBv1Provider) Init() error {
	var err error
	accessKey := os.Getenv(EnvVarAWSAccessKey)
	secretKey := os.Getenv(EnvVarAWSSecretKey)

	p.region = os.Getenv(EnvVarAWSRegion)
	p.vpcID = os.Getenv(EnvVarAWSVpcID)

	if env := os.Getenv(EnvVarUsePrivateIP); len(env) > 0 {
		p.usePrivateIP, err = strconv.ParseBool(env)
		if err != nil {
			return fmt.Errorf("'%s' must be set to a string "+
				"representing a boolean value", EnvVarUsePrivateIP)
		}
	}

	if p.vpcID == "" || p.region == "" {
		p.vpcID, p.region, err = elbv1svc.GetInstanceInfo()
		if err != nil {
			return err
		}
	}

	logrus.Debugf("Initialized provider: region: %s, vpc: %s, usePrivateIP %t",
		p.region, p.vpcID, p.usePrivateIP)

	p.svc, err = elbv1svc.NewService(accessKey, secretKey, p.region, p.vpcID)
	if err != nil {
		return err
	}

	if err := p.svc.CheckAPIConnection(); err != nil {
		return fmt.Errorf("AWS API connection check failed: %v", err)
	}

	logrus.Infof("Configured %s provider in region %s and VPC %s",
		p.GetName(), p.region, p.vpcID)

	return nil
}

/*
 * Methods implementing the providers.Provider interface
 */

func (*AWSELBv1Provider) GetName() string {
	return ProviderName
}

func (p *AWSELBv1Provider) HealthCheck() error {
	return p.svc.CheckAPIConnection()
}

func (p *AWSELBv1Provider) GetLBConfigs() ([]model.LBConfig, error) {
	logrus.Debugf("GetLBConfigs =>")

	var lbConfigs []model.LBConfig
	allLb, err := p.svc.GetLoadBalancers()
	if err != nil {
		return lbConfigs, fmt.Errorf("Failed to lookup load balancers: %v", err)
	}

	logrus.Debugf("GetLBConfigs => found %d load balancers", len(allLb))
	if len(allLb) == 0 {
		return lbConfigs, nil
	}

	allNames := make([]string, len(allLb))
	for i, lb := range allLb {
		allNames[i] = *lb.LoadBalancerName
	}

	lbTags, err := p.svc.DescribeLBTags(allNames)
	if err != nil {
		return lbConfigs, fmt.Errorf("Failed to lookup load balancer tags: %v", err)
	}

	for _, lb := range allLb {
		if _, ok := lbTags[*lb.LoadBalancerName]; !ok {
			continue
		}

		tags := lbTags[*lb.LoadBalancerName]

		var targetPoolName, servicePort string
		var ok bool
		if targetPoolName, ok = tags[TagNameTargetPool]; !ok {
			logrus.Debugf("Skipping LB without targetPool tag: %s", *lb.LoadBalancerName)
			continue
		}
		if servicePort, ok = tags[TagNameServicePort]; !ok {
			logrus.Debugf("Skipping LB without servicePort tag: %s", *lb.LoadBalancerName)
			continue
		}

		lbConfig := model.LBConfig{}
		lbConfig.LBEndpoint = *lb.LoadBalancerName
		lbConfig.LBTargetPoolName = targetPoolName
		var targets []model.LBTarget

		// get currently registered backend instances
		instanceIds, err := p.svc.GetRegisteredInstances(*lb.LoadBalancerName)
		if err != nil {
			return lbConfigs, fmt.Errorf("Failed to get registered instance IDs: %v", err)
		}

		if len(instanceIds) > 0 {
			ec2Instances, err := p.svc.GetInstancesByID(instanceIds)
			if err != nil {
				return lbConfigs, fmt.Errorf("Failed to lookup EC2 instances: %v", err)
			}
			for _, in := range ec2Instances {
				var ip string
				if p.usePrivateIP {
					ip = in.PrivateIPAddress
				} else {
					ip = in.PublicIPAddress
				}

				target := model.LBTarget{
					HostIP: ip,
					Port:   servicePort,
				}
				targets = append(targets, target)
			}
		}

		lbConfig.LBTargets = targets
		lbConfigs = append(lbConfigs, lbConfig)
	}

	logrus.Debugf("GetLBConfigs => Returning %d LB configs", len(lbConfigs))
	return lbConfigs, nil
}

func (p *AWSELBv1Provider) AddLBConfig(config model.LBConfig) (string, error) {
	logrus.Debugf("AddLBConfig => config: %v", config)

	lb, err := p.svc.GetLoadBalancerByName(config.LBEndpoint)
	if err != nil {
		return "", err
	}
	if lb == nil {
		return "", fmt.Errorf("Could not find ELB named '%s'", config.LBEndpoint)
	}

	ec2InstanceIds, err := p.getEC2Instances(config.LBTargets)
	if err != nil {
		return "", fmt.Errorf("Failed to get EC2 instances: %v", err)
	}

	// update the ELB backend instances
	if err := p.ensureBackendInstances(config.LBEndpoint, ec2InstanceIds); err != nil {
		return "", fmt.Errorf("Failed to ensure registered instances: %v", err)
	}

	// tag the ELB
	tags := map[string]string{
		TagNameTargetPool:  config.LBTargetPoolName,
		TagNameServicePort: config.LBTargetPort,
	}

	if err := p.svc.AddLBTags(config.LBEndpoint, tags); err != nil {
		return "", fmt.Errorf("Failed to tag ELB: %v", err)
	}

	logrus.Debug("GetLBConfigs => Done")
	return *lb.DNSName, nil
}

func (p *AWSELBv1Provider) UpdateLBConfig(config model.LBConfig) (string, error) {
	logrus.Debugf("UpdateLBConfig => config: %v", config)

	lb, err := p.svc.GetLoadBalancerByName(config.LBEndpoint)
	if err != nil {
		return "", err
	}
	if lb == nil {
		return "", fmt.Errorf("Could not find ELB named '%s'", config.LBEndpoint)
	}

	ec2InstanceIds, err := p.getEC2Instances(config.LBTargets)
	if err != nil {
		return "", fmt.Errorf("Failed to get EC2 instances: %v", err)
	}

	// update the ELB instances
	if err := p.ensureBackendInstances(config.LBEndpoint, ec2InstanceIds); err != nil {
		return "", fmt.Errorf("Failed to ensure registered instances on ELB %s: %v",
			config.LBEndpoint, err)
	}

	// update the tags
	tags := map[string]string{
		TagNameTargetPool:  config.LBTargetPoolName,
		TagNameServicePort: config.LBTargetPort,
	}

	if err := p.svc.AddLBTags(config.LBEndpoint, tags); err != nil {
		return "", fmt.Errorf("Failed to update servicePort tag: %v", err)
	}

	logrus.Debug("UpdateLBConfig => Done!")
	return *lb.DNSName, nil
}

func (p *AWSELBv1Provider) RemoveLBConfig(config model.LBConfig) error {
	logrus.Debugf("RemoveLBConfig => config: %v", config)

	lb, err := p.svc.GetLoadBalancerByName(config.LBEndpoint)
	if err != nil {
		return err
	}
	if lb == nil {
		return fmt.Errorf("Could not find ELB load balancer named '%s'", config.LBEndpoint)
	}

	// Remove all instances
	if err := p.ensureBackendInstances(*lb.LoadBalancerName, nil); err != nil {
		return fmt.Errorf("Failed to clean up registered instances: %v", err)
	}

	// Remove tag
	if err := p.svc.RemoveLBTag(*lb.LoadBalancerName, TagNameTargetPool); err != nil {
		return fmt.Errorf("Failed to remove targetPool tag: %v", err)
	}

	if err := p.svc.RemoveLBTag(*lb.LoadBalancerName, TagNameServicePort); err != nil {
		return fmt.Errorf("Failed to remove servicePort tag: %v", err)
	}

	logrus.Debug("RemoveLBConfigs => Done")
	return nil
}

/*
 * Private methods
 */

// makes sure the specified instances are registered with specified the load balancer
func (p *AWSELBv1Provider) ensureBackendInstances(loadBalancerName string, instanceIds []string) error {
	logrus.Debugf("ensureBackendInstances => lb: %s, instanceIds: %v", loadBalancerName, instanceIds)
	registeredInstanceIds, err := p.svc.GetRegisteredInstances(loadBalancerName)
	if err != nil {
		return err
	}

	toRegister := differenceStringSlice(instanceIds, registeredInstanceIds)
	toDeregister := differenceStringSlice(registeredInstanceIds, instanceIds)
	logrus.Debugf("Registering instances to ELB %s: %v", loadBalancerName, toRegister)
	logrus.Debugf("Deregistering instances from ELB %s: %v", loadBalancerName, toDeregister)

	if len(toRegister) > 0 {
		if err := p.svc.RegisterInstances(loadBalancerName, toRegister); err != nil {
			return err
		}
	}

	if len(toDeregister) > 0 {
		if err := p.svc.DeregisterInstances(loadBalancerName, toDeregister); err != nil {
			return err
		}
	}

	return nil
}

// looks up the EC2 instances for each of the HostIP in the specified model.LBTarget slice
// and returns their IDs.
func (p *AWSELBv1Provider) getEC2Instances(targets []model.LBTarget) ([]string, error) {
	var instanceIds []string
	var targetIps []string
	for _, t := range targets {
		targetIps = append(targetIps, t.HostIP)
	}

	targetIps = removeDuplicates(targetIps)
	if len(targetIps) == 0 {
		return instanceIds, nil
	}

	ec2Instances, err := p.svc.LookupInstancesByIPAddress(targetIps, p.usePrivateIP)
	if err != nil {
		return instanceIds, err
	}

	logrus.Debugf("getEC2Instances => Looked up %d IP addresses, got %d instances",
		len(targetIps), len(ec2Instances))

	for _, instance := range ec2Instances {
		instanceIds = append(instanceIds, instance.ID)
	}

	return instanceIds, nil
}
