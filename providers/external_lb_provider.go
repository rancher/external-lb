package providers

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/rancher/external-lb/model"
)

type Provider interface {
	// GetName returns the name of the provider.
	GetName() string
	// Init initializes the provider.
	Init() error
	// HealthCheck checks the connection to the provider.
	HealthCheck() error
	// AddLBConfig adds a new endpoint configuration. It may
	// return the FQDN for the endpoint if supported by the provider.
	AddLBConfig(config model.LBConfig) (fqdn string, err error)
	// UpdateLBConfig updates the endpoint configuration. It may
	// return the FQDN for the endpoint if supported by the provider.
	UpdateLBConfig(config model.LBConfig) (fqdn string, err error)
	// RemoveLBConfig removes the specified endpoint configuration.
	RemoveLBConfig(config model.LBConfig) error
	// GetLBConfigs returns all endpoint configurations.
	GetLBConfigs() ([]model.LBConfig, error)
}

var (
	providers = make(map[string]Provider)
)

func GetProvider(name string) (Provider, error) {
	if provider, ok := providers[name]; ok {
		if err := provider.Init(); err != nil {
			return nil, err
		}
		return provider, nil
	}
	return nil, fmt.Errorf("No such provider: %s", name)
}

func RegisterProvider(name string, provider Provider) {
	if _, exists := providers[name]; exists {
		logrus.Fatalf("Provider '%s' tried to register twice", name)
	}
	providers[name] = provider
}
