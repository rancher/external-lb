package providers

import (
	"fmt"
	"github.com/rancher/external-lb/model"
)

type Provider interface {
	GetName() string
	AddLBConfig(config model.LBConfig) error
	RemoveLBConfig(config model.LBConfig) error
	UpdateLBConfig(config model.LBConfig) error
	GetLBConfigs() ([]model.LBConfig, error)
	TestConnection() error
}

var (
	providers map[string]Provider
)

func GetProvider(name string) Provider {
	if provider, ok := providers[name]; ok {
		return provider
	}
	return providers["f5"]
}

func RegisterProvider(name string, provider Provider) error {
	if providers == nil {
		providers = make(map[string]Provider)
	}
	if _, exists := providers[name]; exists {
		return fmt.Errorf("provider %s already registered", name)
	}
	providers[name] = provider
	return nil
}
