package main

import (
	"fmt"
	"os"

	"github.com/rancher/go-rancher/client"
)

type CattleClient struct {
	rancherClient *client.RancherClient
}

func NewCattleClientFromEnvironment() (*CattleClient, error) {
	var cattleURL string
	var cattleAccessKey string
	var cattleSecretKey string

	if env := os.Getenv("CATTLE_URL"); len(env) > 0 {
		cattleURL = env
	} else {
		return nil, fmt.Errorf("Environment variable 'CATTLE_URL' is not set")
	}

	if env := os.Getenv("CATTLE_ACCESS_KEY"); len(env) > 0 {
		cattleAccessKey = env
	} else {
		return nil, fmt.Errorf("Environment variable 'CATTLE_ACCESS_KEY' is not set")
	}

	if env := os.Getenv("CATTLE_SECRET_KEY"); len(env) > 0 {
		cattleSecretKey = env
	} else {
		return nil, fmt.Errorf("Environment variable 'CATTLE_SECRET_KEY' is not set")
	}

	apiClient, err := client.NewRancherClient(&client.ClientOpts{
		Url:       cattleURL,
		AccessKey: cattleAccessKey,
		SecretKey: cattleSecretKey,
	})

	if err != nil {
		return nil, err
	}

	return &CattleClient{
		rancherClient: apiClient,
	}, nil
}

func (c *CattleClient) UpdateServiceFqdn(serviceName, stackName, fqdn string) error {
	event := &client.ExternalDnsEvent{
		EventType:   "dns.update",
		ExternalId:  fqdn,
		ServiceName: serviceName,
		StackName:   stackName,
		Fqdn:        fqdn,
	}
	_, err := c.rancherClient.ExternalDnsEvent.Create(event)
	return err
}

func (c *CattleClient) TestConnect() error {
	opts := &client.ListOpts{}
	_, err := c.rancherClient.ExternalDnsEvent.List(opts)
	return err
}
