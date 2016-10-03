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
	metadataUrl                = "http://rancher-metadata/2015-12-19"
	serviceLabelEndpoint       = "io.rancher.service.external_lb.endpoint"
	serviceLabelEndpointLegacy = "io.rancher.service.external_lb_endpoint"
)

type MetadataClient struct {
	MetadataClient  metadata.Client
	EnvironmentUUID string
}

func getEnvironmentUUID(m metadata.Client) (string, error) {
	timeout := 30 * time.Second
	var err error
	var stack metadata.Stack
	for i := 1 * time.Second; i < timeout; i *= time.Duration(2) {
		stack, err = m.GetSelfStack()
		if err != nil {
			logrus.Errorf("Error reading stack info: %v...will retry", err)
			time.Sleep(i)
		} else {
			return stack.EnvironmentUUID, nil
		}
	}

	return "", fmt.Errorf("Error reading stack info: %v", err)
}

func NewMetadataClient() (*MetadataClient, error) {
	logrus.Debug("Initializing rancher-metadata client")
	m, err := metadata.NewClientAndWait(metadataUrl)
	if err != nil {
		return nil, err
	}

	envUUID, err := getEnvironmentUUID(m)
	if err != nil {
		return nil, fmt.Errorf("Error reading stack metadata info: %v", err)
	}

	return &MetadataClient{
		MetadataClient:  m,
		EnvironmentUUID: envUUID,
	}, nil
}

func (m *MetadataClient) GetVersion() (string, error) {
	return m.MetadataClient.GetVersion()
}

func (m *MetadataClient) GetMetadataLBConfigs(targetPoolSuffix string) (map[string]model.LBConfig, error) {
	lbConfigs := make(map[string]model.LBConfig)
	services, err := m.MetadataClient.GetServices()
	if err != nil {
		logrus.Infof("Error reading services: %v", err)
	} else {
		for _, service := range services {
			endpoint, ok := service.Labels[serviceLabelEndpoint]
			if !ok {
				endpoint, ok = service.Labels[serviceLabelEndpointLegacy]
			}
			if !ok {
				continue
			}

			logrus.Debugf("LB label exists for service : %v", service.Name)
			// Configure this service only if this endpoint is already not used by some other service so far
			_, ok = lbConfigs[endpoint]
			if ok {
				logrus.Errorf("Endpoint %s already used by another service, will skip this service : %s",
					endpoint, service.Name)
				continue
			}

			// get the service port
			if len(service.Ports) == 0 {
				logrus.Warnf("Skipping LB configuration for service %s: "+
					"Service hasn't any ports exposed", service.Name)
				continue
			}

			portspec := strings.Split(service.Ports[0], ":")
			if len(portspec) != 2 {
				logrus.Warnf("Skipping LB configuration for service %s: "+
					"Unexpected format of service port spec: %s",
					service.Name, service.Ports[0])
				continue
			}

			lbConfig := model.LBConfig{}
			lbConfig.LBEndpoint = endpoint
			lbConfig.LBTargetPort = portspec[0]
			lbConfig.LBTargetPoolName = fmt.Sprintf("%s_%s_%s_%s", service.Name, service.StackName,
				m.EnvironmentUUID, targetPoolSuffix)

			if err = m.getContainerLBTargets(&lbConfig, service); err != nil {
				continue
			}

			lbConfigs[endpoint] = lbConfig
		}
	}

	return lbConfigs, nil
}

func (m *MetadataClient) getContainerLBTargets(lbConfig *model.LBConfig, service metadata.Service) error {
	for _, container := range service.Containers {
		if len(container.ServiceName) == 0 {
			continue
		}

		if !containerStateOK(container) {
			logrus.Debugf("Skipping container %s with state '%s' and health '%s'",
				container.Name, container.State, container.HealthState)
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

		for _, port := range container.Ports {
			// split the container port to get the publicip:port
			portspec := strings.Split(port, ":")
			if len(portspec) != 3 {
				logrus.Warnf("Unexpected format of port spec for container %s: %s", container.Name, port)
				continue
			}

			ip := portspec[0]
			port := portspec[1]
			if port != lbConfig.LBTargetPort {
				logrus.Debugf("Container portspec '%s' does not match LBTargetPort %s", portspec, lbConfig.LBTargetPort)
				continue
			}

			lbTarget := model.LBTarget{
				HostIP: ip,
				Port:   port,
			}
			lbConfig.LBTargets = append(lbConfig.LBTargets, lbTarget)
		}
	}

	logrus.Debugf("Found %d target IPs for service %s", len(lbConfig.LBTargets), service.Name)
	return nil
}

func containerStateOK(container metadata.Container) bool {
	switch container.State {
	case "running":
	default:
		return false
	}

	switch container.HealthState {
	case "healthy":
	case "updating-healthy":
	case "":
	default:
		return false
	}

	return true
}
