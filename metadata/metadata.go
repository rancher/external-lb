package metadata

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/rancher/external-lb/model"
	"github.com/rancher/go-rancher-metadata/metadata"
	"time"
	"strings"
)

const (
	metadataUrl = "http://localhost:90/latest"
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

func (m *MetadataClient) GetMetadataLBRecords() (map[string]model.LBRecord, error) {
	lbRecords := make(map[string]model.LBRecord)

	services, err := m.MetadataClient.GetServices()
	
	if err != nil {
		logrus.Infof("Error reading services %v", err)
	} else {
		for _, service := range services {
			vip, ok := service.Labels["io.rancher.service.external_lb_vip"]
			if ok {
				//label exists, configure external LB
				logrus.Debugf("label exists for service : %v", service)
				lbRecord := model.LBRecord{}
				lbRecord.Vip = vip
				lbRecord.ServiceName = service.Name
				err = m.getContainerLBNodes(&lbRecord, service)
				if err != nil {
					continue
				}
				lbRecords[service.Name] = lbRecord
			} else {
				continue
			}
		}
	}
	
	return lbRecords, nil
}

func (m *MetadataClient) getContainerLBNodes(lbRecord *model.LBRecord, service metadata.Service) error {
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
		
			lbNode := model.LBNode{}
			lbNode.HostIP = ip
			lbNode.Port = port
			lbRecord.Nodes = append(lbRecord.Nodes, lbNode)
		} else {
			logrus.Debugf("Skipping container, PortSpec for container does not have host_ip:public_port:private:port format, container: %s, service: %s, ports: %s ", container.Name, container.ServiceName, container.Ports)
		}
	}

	return nil
}
