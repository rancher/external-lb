package metadata

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/rancher/external-lb/model"
	"github.com/rancher/go-rancher-metadata/metadata"
	"strings"
	"time"
)

const (
	metadataUrl = "http://rancher-metadata/latest"
)

type MetadataClient struct {
	MetadataClient  *metadata.Client
	EnvironmentName string
}

func getEnvironmentName(m *metadata.Client) (string, error) {
	timeout := 30 * time.Second
	var err error
	var stack metadata.Stack
	for i := 1 * time.Second; i < timeout; i *= time.Duration(2) {
		stack, err = m.GetSelfStack()
		if err != nil {
			logrus.Errorf("Error reading stack info: %v...will retry", err)
			time.Sleep(i)
		} else {
			return stack.EnvironmentName, nil
		}
	}
	return "", fmt.Errorf("Error reading stack info: %v", err)
}

func NewMetadataClient() (*MetadataClient, error) {
	m, err := metadata.NewClientAndWait(metadataUrl)
	if err != nil {
		logrus.Fatalf("Failed to configure rancher-metadata: %v", err)
	}

	envName, err := getEnvironmentName(m)
	if err != nil {
		logrus.Fatalf("Error reading stack info: %v", err)
	}

	return &MetadataClient{
		MetadataClient:  m,
		EnvironmentName: envName,
	}, nil
}

func (m *MetadataClient) GetVersion() (string, error) {
	return m.MetadataClient.GetVersion()
}

func (m *MetadataClient) GetMetadataLBConfigs() (map[string]model.LBConfig, error) {
	lbConfigs := make(map[string]model.LBConfig)

	services, err := m.MetadataClient.GetServices()

	if err != nil {
		logrus.Infof("Error reading services %v", err)
	} else {
		for _, service := range services {
			lb_endpoint, ok := service.Labels["io.rancher.service.external_lb_endpoint"]
			if ok {
				//label exists, configure external LB
				logrus.Debugf("label exists for service : %v", service)
				lbConfig := model.LBConfig{}
				lbConfig.LBEndpoint = lb_endpoint
				lbConfig.LBTargetName = service.StackName + "." + service.Name + "_rancher.internal"
				err = m.getContainerLBTargets(&lbConfig, service)
				if err != nil {
					continue
				}
				lbConfigs[lb_endpoint] = lbConfig
			} else {
				continue
			}
		}
	}

	return lbConfigs, nil
}

func (m *MetadataClient) getContainerLBTargets(lbConfig *model.LBConfig, service metadata.Service) error {
	containers := service.Containers

	for _, container := range containers {
		if len(container.ServiceName) == 0 {
			continue
		}

		if len(service.Name) != 0 {
			if service.Name != container.ServiceName {
				continue
			}
			if service.StackName != container.StackName {
				continue
			}
		}

		if len(container.Ports) == 0 {
			continue
		}

		//split the container.Ports to get the publicip:port
		portspec := strings.Split(container.Ports[0], ":")

		if len(portspec) > 2 {
			ip := portspec[0]
			port := portspec[1]

			lbTarget := model.LBTarget{}
			lbTarget.HostIP = ip
			lbTarget.Port = port
			lbConfig.LBTargets = append(lbConfig.LBTargets, lbTarget)
		} else {
			logrus.Debugf("Skipping container, PortSpec for container does not have host_ip:public_port:private:port format, container: %s, service: %s, ports: %s ", container.Name, container.ServiceName, container.Ports)
		}
	}

	return nil
}
