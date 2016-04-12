package providers

import (
	"fmt"
	//"github.com/Sirupsen/logrus"
	"github.com/rancher/external-lb/model"
)

type Provider interface {
	GetName() string
	ConfigureLBRecord(model.LBRecord) error
}

var (
	providers map[string]Provider
)

func init() {
	// try to resolve rancher-metadata before going further
	// the resolution indicates that the network has been set

}

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
		return fmt.Errorf("provider already registered")
	}
	providers[name] = provider
	return nil
}
