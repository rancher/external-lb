package slb

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/Sirupsen/logrus"
	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/ecs"
	"github.com/denverdino/aliyungo/slb"
	"github.com/rancher/external-lb/model"
	"github.com/rancher/external-lb/providers"
)

const (
	ProviderName = "Aliyun SLB"
	ProviderSlug = "aliyun_slb"
)

const (
	TagResourceFlag    = "external-lb/rancher"
	TagNameTargetPool  = "external-lb/targetPoolName"
	TagNameServicePort = "external-lb/servicePort"
)

const (
	EnvVarSLBAccessKey = "SLB_ACCESS_KEY"
	EnvVarSLBSecretKey = "SLB_SECRET_KEY"
	EnvVarSLBRegionId  = "SLB_REGION_ID"
	EnvVarSLBVpcId     = "SLB_VPC_ID"
	EnvVarUsePrivateIP = "SLB_USE_PRIVATE_IP"
)

const (
	DefaultBackendServerWeight = 100
)

type AliyunSLBProvider struct {
	slbClient *slb.Client
	ecsClient *ecs.Client

	vpcId        string
	regionId     string
	usePrivateIP bool
}

func init() {
	providers.RegisterProvider(ProviderSlug, new(AliyunSLBProvider))
}

func (p *AliyunSLBProvider) Init() error {
	var err error
	accessKeyId := os.Getenv(EnvVarSLBAccessKey)
	if len(accessKeyId) == 0 {
		return fmt.Errorf("%s is not set", EnvVarSLBAccessKey)
	}
	accessKeySecret := os.Getenv(EnvVarSLBSecretKey)
	if len(accessKeySecret) == 0 {
		return fmt.Errorf("%s is not set", EnvVarSLBSecretKey)
	}
	p.regionId = os.Getenv(EnvVarSLBRegionId)
	if len(p.regionId) == 0 {
		return fmt.Errorf("%s is not set", EnvVarSLBRegionId)
	}
	p.vpcId = os.Getenv(EnvVarSLBVpcId)
	if env := os.Getenv(EnvVarUsePrivateIP); len(env) > 0 {
		p.usePrivateIP, err = strconv.ParseBool(env)
		if err != nil {
			return fmt.Errorf("'%s' must be set to a string "+
				"representing a boolean value", EnvVarUsePrivateIP)
		}
	}
	logrus.Debugf("Initializing %s provider", p.GetName())

	p.slbClient = slb.NewSLBClient(accessKeyId, accessKeySecret, common.Region(p.regionId))
	p.ecsClient = ecs.NewECSClient(accessKeyId, accessKeySecret, common.Region(p.regionId))

	if err := p.HealthCheck(); err != nil {
		return fmt.Errorf("Cloud not connect SLB endpoint")
	}

	logrus.Infof("Configured %s provider in region %s and VPC %s",
		p.GetName(), p.regionId, p.vpcId)

	return nil
}

func (p *AliyunSLBProvider) GetName() string {
	return ProviderName
}

func (p *AliyunSLBProvider) HealthCheck() error {
	_, err := p.slbClient.DescribeRegions()
	if err != nil {
		return fmt.Errorf("Failed to list SLB available regions: %v", err)
	}
	return nil
}

func (p *AliyunSLBProvider) AddLBConfig(config model.LBConfig) (string, error) {
	logrus.Debugf("AddLBConfig => config: %v", config)
	lb, err := p.getLoadBalancerById(config.LBEndpoint)
	if err != nil {
		return "", err
	}

	if !p.checkListenersInstancePort(config.LBEndpoint, config.LBTargetPort) {
		return "", fmt.Errorf(
			"SLB '%s' is not configured with instance port matching the service port '%s'",
			config.LBEndpoint, config.LBTargetPort)
	}

	ecsInstanceIds, err := p.getECSInstances(config.LBTargets)
	if err != nil {
		return "", fmt.Errorf("Failed to get ECS instances: %v", err)
	}

	// register SLB instances
	if err := p.ensureBackendInstances(lb.LoadBalancerId, ecsInstanceIds); err != nil {
		return "", fmt.Errorf("Failed to ensure registered instances: %v", err)
	}

	// add ELB tags
	addTagArgs := &slb.AddTagsArgs{
		RegionId:       common.Region(p.regionId),
		LoadBalancerID: lb.LoadBalancerId,
		Tags:           getLBTags(config),
	}
	if err := p.slbClient.AddTags(addTagArgs); err != nil {
		return "", fmt.Errorf("Failed to tag ELB: %v", err)
	}

	logrus.Debug("AddLBConfigs => Done")
	return lb.Address, nil
}

func (p *AliyunSLBProvider) RemoveLBConfig(config model.LBConfig) error {
	logrus.Debugf("RemoveLBConfig => config: %v", config)
	lb, err := p.getLoadBalancerById(config.LBEndpoint)
	if err != nil {
		return err
	}

	// deregister all instances
	if err := p.ensureBackendInstances(lb.LoadBalancerId, nil); err != nil {
		return fmt.Errorf("Failed to clean up registered instances: %v", err)
	}

	// remove ELB tags
	args := &slb.RemoveTagsArgs{
		RegionId:       common.Region(p.regionId),
		LoadBalancerID: lb.LoadBalancerId,
		Tags:           getLBTags(config),
	}
	err = p.slbClient.RemoveTags(args)
	if err != nil {
		return err
	}

	logrus.Debug("RemoveLBConfigs => Done")
	return nil
}

func (p *AliyunSLBProvider) UpdateLBConfig(config model.LBConfig) (string, error) {
	logrus.Debugf("UpdateLBConfig => config: %v", config)
	lb, err := p.getLoadBalancerById(config.LBEndpoint)
	if err != nil {
		return "", err
	}

	if !p.checkListenersInstancePort(config.LBEndpoint, config.LBTargetPort) {
		return "", fmt.Errorf(
			"SLB '%s' is not configured with instance port matching the service port '%s'",
			config.LBEndpoint, config.LBTargetPort)
	}

	ecsInstanceIds, err := p.getECSInstances(config.LBTargets)
	if err != nil {
		return "", fmt.Errorf("Failed to get ECS instances: %v", err)
	}

	// update SLB instances
	if err := p.ensureBackendInstances(lb.LoadBalancerId, ecsInstanceIds); err != nil {
		return "", fmt.Errorf("Failed to ensure registered instances: %v", err)
	}

	// update SLB tags
	addTagArgs := &slb.AddTagsArgs{
		RegionId:       common.Region(p.regionId),
		LoadBalancerID: lb.LoadBalancerId,
		Tags:           getLBTags(config),
	}

	if err := p.slbClient.AddTags(addTagArgs); err != nil {
		return "", fmt.Errorf("Failed to tag ELB: %v", err)
	}

	logrus.Debug("UpdateLBConfig => Done!")
	return lb.Address, nil
}

func (p *AliyunSLBProvider) GetLBConfigs() ([]model.LBConfig, error) {
	logrus.Debugf("GetLBConfigs =>")

	var lbConfigs []model.LBConfig

	args := &slb.DescribeLoadBalancersArgs{
		RegionId: common.Region(p.regionId),
	}
	if len(p.vpcId) > 0 {
		args.VpcId = p.vpcId
		args.NetworkType = string(common.VPC)
	} else {
		args.NetworkType = string(common.Classic)
	}
	allLb, err := p.slbClient.DescribeLoadBalancers(args)
	if err != nil {
		return lbConfigs, fmt.Errorf("Failed to lookup load balancers: %v", err)
	}

	logrus.Debugf("GetLBConfigs => found %d load balancers", len(allLb))
	if len(allLb) == 0 {
		return lbConfigs, nil
	}

	for _, lb := range allLb {
		tagArgs := &slb.DescribeTagsArgs{
			RegionId:       common.Region(p.regionId),
			LoadBalancerID: lb.LoadBalancerId,
		}
		tags, _, err := p.slbClient.DescribeTags(tagArgs)
		if err != nil {
			return lbConfigs, err
		}

		var targetPoolName, servicePort string
		var ok bool
		for _, tag := range tags {
			if tag.TagKey == TagResourceFlag {
				ok = true
			}
			if tag.TagKey == TagNameTargetPool {
				targetPoolName = tag.TagValue
			}
			if tag.TagKey == TagNameServicePort {
				servicePort = tag.TagValue
			}
		}
		if !ok {
			logrus.Debugf("Skipping LB without RancherResource tag: %s", lb.LoadBalancerId)
			continue
		}

		lbConfig := model.LBConfig{}
		lbConfig.LBEndpoint = lb.LoadBalancerName
		lbConfig.LBTargetPoolName = targetPoolName

		var targets []model.LBTarget
		var registeredInstanceIds []string

		// get currently registered backend instances
		hsArgs := &slb.DescribeHealthStatusArgs{
			LoadBalancerId: lb.LoadBalancerId,
		}
		healthStatusResonpse, err := p.slbClient.DescribeHealthStatus(hsArgs)
		if err != nil {
			return lbConfigs, fmt.Errorf("Failed to get registered instance IDs: %v", err)
		}
		for _, hst := range healthStatusResonpse.BackendServers.BackendServer {
			registeredInstanceIds = append(registeredInstanceIds, hst.ServerId)
		}

		if len(registeredInstanceIds) > 0 {
			instanceIdsStr, _ := json.Marshal(registeredInstanceIds)
			instArgs := &ecs.DescribeInstancesArgs{
				RegionId:    common.Region(p.regionId),
				InstanceIds: string(instanceIdsStr),
			}
			ecsInstances, _, err := p.ecsClient.DescribeInstances(instArgs)
			if err != nil {
				return lbConfigs, err
			}
			for _, inst := range ecsInstances {
				var ip string
				if p.usePrivateIP {
					ip = inst.InnerIpAddress.IpAddress[0]
				} else {
					ip = inst.PublicIpAddress.IpAddress[0]
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

func (p *AliyunSLBProvider) getLoadBalancerById(loadBalancerId string) (slb.LoadBalancerType, error) {
	var lb slb.LoadBalancerType
	args := &slb.DescribeLoadBalancersArgs{
		RegionId:       common.Region(p.regionId),
		LoadBalancerId: loadBalancerId,
	}
	lbs, err := p.slbClient.DescribeLoadBalancers(args)
	if err != nil {
		return lb, err
	}
	if lbs == nil || len(lbs) == 0 {
		return lb, fmt.Errorf("Could not find SLB id: '%s'", loadBalancerId)
	}
	return lbs[0], nil
}

func (p *AliyunSLBProvider) ensureBackendInstances(loadBalancerId string, instanceIds []string) error {
	logrus.Debugf("ensureBackendInstances => lb: %s, instanceIds: %v", loadBalancerId, instanceIds)
	var registeredInstanceIds []string

	args := &slb.DescribeHealthStatusArgs{
		LoadBalancerId: loadBalancerId,
	}
	healthStatusResonpse, err := p.slbClient.DescribeHealthStatus(args)
	if err != nil {
		return err
	}
	for _, hst := range healthStatusResonpse.BackendServers.BackendServer {
		registeredInstanceIds = append(registeredInstanceIds, hst.ServerId)
	}

	toRegister := differenceStringSlice(instanceIds, registeredInstanceIds)
	toDeregister := differenceStringSlice(registeredInstanceIds, instanceIds)
	logrus.Debugf("Registering instances to SLB %s: %v", loadBalancerId, toRegister)
	logrus.Debugf("Deregistering instances from SLB %s: %v", loadBalancerId, toDeregister)

	if len(toRegister) > 0 {
		var toRegisterBS []slb.BackendServerType
		for _, id := range toRegister {
			bs := slb.BackendServerType{ServerId: id, Weight: DefaultBackendServerWeight}
			toRegisterBS = append(toRegisterBS, bs)
		}
		_, err := p.slbClient.AddBackendServers(loadBalancerId, toRegisterBS)
		if err != nil {
			return err
		}
	}

	if len(toDeregister) > 0 {
		_, err := p.slbClient.RemoveBackendServers(loadBalancerId, toDeregister)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *AliyunSLBProvider) getECSInstances(targets []model.LBTarget) ([]string, error) {
	var instanceIds []string
	var targetIps []string
	for _, t := range targets {
		targetIps = append(targetIps, t.HostIP)
	}

	targetIps = removeDuplicates(targetIps)
	if len(targetIps) == 0 {
		return instanceIds, nil
	}

	args := &ecs.DescribeInstancesArgs{
		RegionId: common.Region(p.regionId),
	}
	ipAddresses, _ := json.Marshal(targetIps)
	logrus.Debugf("getECSInstances: ipAddresses %s", string(ipAddresses))
	if len(p.vpcId) == 0 {
		args.InstanceNetworkType = string(common.Classic)
		if p.usePrivateIP {
			args.InnerIpAddresses = string(ipAddresses)
		}
	} else {
		args.InstanceNetworkType = string(common.VPC)
		args.VpcId = p.vpcId
		if p.usePrivateIP {
			args.PrivateIpAddresses = string(ipAddresses)
		}
	}
	if !p.usePrivateIP {
		args.PublicIpAddresses = string(ipAddresses)
	}

	ecsInstances, _, err := p.ecsClient.DescribeInstances(args)
	if err != nil {
		return instanceIds, err
	}

	logrus.Debugf("getECSInstances => Looked up %d IP addresses, got %d instances",
		len(targetIps), len(ecsInstances))

	for _, instance := range ecsInstances {
		instanceIds = append(instanceIds, instance.InstanceId)
	}

	return instanceIds, nil
}

func (p *AliyunSLBProvider) checkListenersInstancePort(loadBalancerId string, port string) bool {
	found := false

	lb, err := p.slbClient.DescribeLoadBalancerAttribute(loadBalancerId)
	if err != nil {
		logrus.Errorf("Failed to DescribeLoadBalancerAttribute: %v", err)
		return found
	}

	logrus.Debugf("SLB: %s listenerPorts: %v", lb.LoadBalancerId, lb.ListenerPorts)
	logrus.Debugf("SLB: %s ListenerPortsAndProtocol: %v", lb.LoadBalancerId, lb.ListenerPortsAndProtocol)
	for _, listenerPort := range lb.ListenerPorts.ListenerPort {
		if strconv.Itoa(listenerPort) == port {
			found = true
			break
		}
	}

	return found
}

func getLBTags(config model.LBConfig) string {
	tags := []map[string]string{
		{"TagKey": TagNameTargetPool, "TagValue": config.LBTargetPoolName},
		{"TagKey": TagNameServicePort, "TagValue": config.LBTargetPort},
		{"TagKey": TagResourceFlag, "TagValue": "Y"},
	}
	tagsBytes, err := json.Marshal(tags)
	if err != nil {
		logrus.Debugf("Failed to marshal tags json: %v", err)
		return ""
	}
	return string(tagsBytes)
}

func removeDuplicates(in []string) (out []string) {
	m := map[string]bool{}
	for _, v := range in {
		if _, found := m[v]; !found {
			out = append(out, v)
			m[v] = true
		}
	}
	return out
}

func differenceStringSlice(sliceA []string, sliceB []string) []string {
	var diff []string

	for _, a := range sliceA {
		found := false
		for _, b := range sliceB {
			if a == b {
				found = true
				break
			}
		}
		if !found {
			diff = append(diff, a)
		}
	}

	return diff
}
