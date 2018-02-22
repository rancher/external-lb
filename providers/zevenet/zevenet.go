package zevenet

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	zlb "github.com/konsorten/zevenet-lb-go"
	"github.com/rancher/external-lb/model"
	"github.com/rancher/external-lb/providers"
)

const (
	providerName = "Zevenet"
	providerSlug = "zevenet"
)

type ZevenetProvider struct {
	client   *zlb.ZapiSession
	farmName string
}

func init() {
	providers.RegisterProvider(providerSlug, new(ZevenetProvider))
}

func getServiceName(config *model.LBConfig) (pn string) {
	// format: <service_name>_<environment_uuid>_rancher.internal
	pn = config.LBTargetPoolName

	// replace invalid chars
	pn = strings.Replace(pn, ".", "--D--", -1)
	pn = strings.Replace(pn, "_", "--U--", -1)

	return
}

func getPoolName(service *zlb.ServiceDetails) (pn string) {
	pn = service.ServiceName

	// replace invalid chars
	pn = strings.Replace(pn, "--D--", ".", -1)
	pn = strings.Replace(pn, "--U--", "_", -1)

	return
}

func (p *ZevenetProvider) Init() (err error) {
	host := os.Getenv("ZAPI_HOST")
	if len(host) == 0 {
		return fmt.Errorf("ZAPI_HOST is not set")
	}

	zapiKey := os.Getenv("ZAPI_KEY")
	if len(zapiKey) == 0 {
		return fmt.Errorf("ZAPI_KEY is not set")
	}

	p.farmName = os.Getenv("ZAPI_FARM")
	if len(p.farmName) == 0 {
		return fmt.Errorf("ZAPI_FARM is not set")
	}

	log.Debugf("Initializing Zevenet provider with farm %v on host: %s, key-length: %d", p.farmName, host, len(zapiKey))

	p.client, err = zlb.Connect(host, zapiKey, nil)

	if err != nil {
		return
	}

	log.Infof("Configured %s provider using farm %v on host %s", p.GetName(), p.farmName, host)
	return
}

func (p *ZevenetProvider) GetName() string {
	return providerName
}

func (p *ZevenetProvider) HealthCheck() error {
	success, msg := p.client.Ping()

	if !success {
		return fmt.Errorf("Failed to ping Zevenet loadbalancer: %v", msg)
	}

	return nil
}

func (p *ZevenetProvider) AddLBConfig(config model.LBConfig) (string, error) {
	// first check if changes can be made
	if available, msg := p.client.Ping(); !available {
		return "", fmt.Errorf("Failed to ping Zevenet loadbalancer: %v", msg)
	}

	// check if the farm exists
	farm, err := p.client.GetFarm(p.farmName)

	if err != nil {
		return "", fmt.Errorf("Failed to get farm from Zevenet loadbalancer: %v", err)
	}

	if farm == nil {
		return "", fmt.Errorf("Farm not found on Zevenet loadbalancer: %v", p.farmName)
	}

	// check if http redirection applies
	httpRedirectURL, _ := config.LBLabels["httpRedirectUrl"]

	if farm.Listener != zlb.FarmListener_HTTP {
		httpRedirectURL = ""
	}

	// delete the service
	serviceName := getServiceName(&config)

	_, err = p.client.DeleteService(farm.FarmName, serviceName)

	if err != nil {
		return "", fmt.Errorf("Failed to delete service on Zevenet loadbalancer: %v", err)
	}

	// re-create the service
	service, err := p.client.CreateService(farm.FarmName, serviceName)

	if err != nil {
		return "", fmt.Errorf("Failed to create service on Zevenet loadbalancer: %v", err)
	}

	// update values
	service.HostPattern = config.LBEndpoint
	service.RedirectURL = httpRedirectURL

	err = p.client.UpdateService(service)

	if err != nil {
		return "", fmt.Errorf("Failed to update service on Zevenet loadbalancer: %v", err)
	}

	// add backends (if not redirecting)
	if httpRedirectURL == "" {
		for _, target := range config.LBTargets {
			// get the port number
			port, err := strconv.Atoi(target.Port)

			if err != nil {
				return "", fmt.Errorf("Failed to parse port number '%v': %v", target.Port, err)
			}

			// create the backend
			_, err = p.client.CreateBackend(farm.FarmName, service.ServiceName, target.HostIP, port)

			if err != nil {
				return "", fmt.Errorf("Failed to create backend on Zevenet loadbalancer: %v", err)
			}
		}
	}

	// restart loadbalancer
	err = p.client.RestartFarm(farm.FarmName)

	if err != nil {
		return "", fmt.Errorf("Failed to restart farm on Zevenet loadbalancer: %v", err)
	}

	return "", nil
}

func (p *ZevenetProvider) UpdateLBConfig(config model.LBConfig) (string, error) {
	return p.AddLBConfig(config)
}

func (p *ZevenetProvider) RemoveLBConfig(config model.LBConfig) error {
	// first check if changes can be made
	if available, msg := p.client.Ping(); !available {
		return fmt.Errorf("Failed to ping Zevenet loadbalancer: %v", msg)
	}

	// check if the farm exists
	farm, err := p.client.GetFarm(p.farmName)

	if err != nil {
		return fmt.Errorf("Failed to get farm from Zevenet loadbalancer: %v", err)
	}

	if farm == nil {
		return fmt.Errorf("Farm not found on Zevenet loadbalancer: %v", p.farmName)
	}

	// delete the service
	serviceName := getServiceName(&config)

	deleted, err := p.client.DeleteService(farm.FarmName, serviceName)

	if err != nil {
		return fmt.Errorf("Failed to delete service from Zevenet loadbalancer: %v", err)
	}

	if !deleted {
		// nothing deleted, skip restart
		return nil
	}

	// restart loadbalancer
	err = p.client.RestartFarm(farm.FarmName)

	if err != nil {
		return fmt.Errorf("Failed to restart farm on Zevenet loadbalancer: %v", err)
	}

	return nil
}

func (p *ZevenetProvider) GetLBConfigs() ([]model.LBConfig, error) {
	// first check if changes can be made
	if available, msg := p.client.Ping(); !available {
		return nil, fmt.Errorf("Failed to ping Zevenet loadbalancer: %v", msg)
	}

	// check if the farm exists
	farm, err := p.client.GetFarm(p.farmName)

	if err != nil {
		return nil, fmt.Errorf("Failed to get farm from Zevenet loadbalancer: %v", err)
	}

	if farm == nil {
		return nil, fmt.Errorf("Farm not found on Zevenet loadbalancer: %v", p.farmName)
	}

	// get services
	var lbConfigs []model.LBConfig

	for _, service := range farm.Services {
		cfg := model.LBConfig{
			LBTargetPoolName: getPoolName(&service),
			LBTargetPort:     strconv.Itoa(farm.VirtualPort),
			LBEndpoint:       service.HostPattern,
		}

		// get backends
		for _, backend := range service.Backends {
			cfg.LBTargets = append(cfg.LBTargets, model.LBTarget{
				HostIP: backend.IPAddress,
				Port:   strconv.Itoa(backend.Port),
			})
		}

		lbConfigs = append(lbConfigs, cfg)
	}

	return lbConfigs, nil
}
