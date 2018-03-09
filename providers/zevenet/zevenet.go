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
	client *zlb.ZapiSession
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

	log.Debugf("Initializing Zevenet provider: %s, key-length: %d", host, len(zapiKey))

	p.client, err = zlb.Connect(host, zapiKey, nil)

	if err != nil {
		return
	}

	log.Infof("Configured %s provider using %s", p.GetName(), host)
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

	// retrieve farm list
	farmList, _ := config.LBLabels["farms"]

	if farmList == "" {
		return "", fmt.Errorf("No farm specified; missing 'io.rancher.service.external_lb.farms' label?")
	}

	farms := strings.Split(farmList, ",")

	// add configurations
	for _, farmName := range farms {
		// ignore empty entry
		if farmName == "" {
			continue
		}

		// configure
		_, err := p.addLBConfigSingleFarm(farmName, config)

		if err != nil {
			log.Errorf("Failed to add farm %v: %v", farmName, err)
		}
	}

	return "", nil
}

func (p *ZevenetProvider) addLBConfigSingleFarm(farmName string, config model.LBConfig) (string, error) {
	// check if the farm exists
	farm, err := p.client.GetFarm(farmName)

	if err != nil {
		return "", fmt.Errorf("Failed to get farm from Zevenet loadbalancer: %v", err)
	}

	if farm == nil {
		return "", fmt.Errorf("Farm not found on Zevenet loadbalancer: %v", farmName)
	}

	// delete the service
	serviceName := getServiceName(&config)

	log.Debugf("Deleting service on farm %v: %v", farm.FarmName, serviceName)

	_, err = p.client.DeleteService(farm.FarmName, serviceName)

	if err != nil {
		return "", fmt.Errorf("Failed to delete service on Zevenet loadbalancer: %v", err)
	}

	// check if http redirection applies
	log.Debugf("Adding service on farm %v: %v", farm.FarmName, serviceName)
	log.Debugf("Service labels: %v", config.LBLabels)

	httpRedirectURL, _ := config.LBLabels["http_redirect_url"]

	if farm.Listener != zlb.FarmListener_HTTP {
		httpRedirectURL = ""
	}

	checkCmd, _ := config.LBLabels["check"]

	// re-create the service
	service, err := p.client.CreateService(farm.FarmName, serviceName)

	if err != nil {
		return "", fmt.Errorf("Failed to create service on Zevenet loadbalancer: %v", err)
	}

	// update values
	service.HostPattern = config.LBEndpoint

	if httpRedirectURL != "" {
		log.Debugf("Setting redirect URL for service '%v': %v", serviceName, httpRedirectURL)

		service.RedirectURL = httpRedirectURL
		service.RedirectType = zlb.ServiceRedirectType_Default
	} else {
		// enable farm guardian
		if checkCmd != "" {
			service.FarmGuardianEnabled = true
			service.FarmGuardianLogsEnabled = zlb.OptionalBool_True
			service.FarmGuardianCheckIntervalSeconds = 5

			if checkCmd == "true" {
				service.FarmGuardianScript = "check_http -H HOST -p PORT"
			} else {
				service.FarmGuardianScript = checkCmd
			}
		}
	}

	err = p.client.UpdateService(service)

	if err != nil {
		return "", fmt.Errorf("Failed to update service on Zevenet loadbalancer: %v", err)
	}

	// add backends (if not redirecting)
	if httpRedirectURL == "" {
		log.Debugf("Adding backends to service: %v", serviceName)

		for _, target := range config.LBTargets {
			// get the port number
			port, err := strconv.Atoi(target.Port)

			if err != nil {
				return "", fmt.Errorf("Failed to parse port number '%v': %v", target.Port, err)
			}

			// create the backend
			log.Debugf("Adding backend to service '%v': %v:%v", serviceName, target.HostIP, port)

			_, err = p.client.CreateBackend(farm.FarmName, service.ServiceName, target.HostIP, port)

			if err != nil {
				return "", fmt.Errorf("Failed to create backend on Zevenet loadbalancer: %v", err)
			}
		}
	}

	// restart loadbalancer
	log.Debugf("Restarting farm: %v", farm.FarmName)

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

	// retrieve farm list
	farmList, _ := config.LBLabels["farms"]

	if farmList == "" {
		return fmt.Errorf("No farm specified; missing 'io.rancher.service.external_lb.farms' label?")
	}

	farms := strings.Split(farmList, ",")
	serviceName := getServiceName(&config)

	for _, farmName := range farms {
		// delete the service
		log.Debugf("Deleting service on farm %v: %v", farmName, serviceName)

		deleted, err := p.client.DeleteService(farmName, serviceName)

		if err != nil {
			return fmt.Errorf("Failed to delete service from Zevenet loadbalancer: %v", err)
		}

		if !deleted {
			// nothing deleted, skip restart
			log.Debugf("Service does not exist on farm %v: %v; Skipping farm restart...", farmName, serviceName)

			return nil
		}

		// restart loadbalancer
		log.Debugf("Restarting farm: %v", farmName)

		err = p.client.RestartFarm(farmName)

		if err != nil {
			return fmt.Errorf("Failed to restart farm on Zevenet loadbalancer: %v", err)
		}
	}

	return nil
}

func (p *ZevenetProvider) GetLBConfigs() ([]model.LBConfig, error) {
	// first check if changes can be made
	if available, msg := p.client.Ping(); !available {
		return nil, fmt.Errorf("Failed to ping Zevenet loadbalancer: %v", msg)
	}

	// get all farms
	farms, err := p.client.GetAllFarms()

	if err != nil {
		return nil, fmt.Errorf("Failed to get farms from Zevenet loadbalancer: %v", err)
	}

	lbConfigMap := make(map[string]model.LBConfig)

	for _, farmInfo := range farms {
		// check if the farm exists
		log.Debugf("Gathering existing services on farm: %v", farmInfo.FarmName)

		farm, err := p.client.GetFarm(farmInfo.FarmName)

		if err != nil {
			return nil, fmt.Errorf("Failed to get farm %v from Zevenet loadbalancer: %v", farmInfo.FarmName, err)
		}

		if farm == nil {
			return nil, fmt.Errorf("Farm not found on Zevenet loadbalancer: %v", farmInfo.FarmName)
		}

		// get services
		for _, service := range farm.Services {
			log.Debugf("Found service on farm '%v': %v", farm.FarmName, service.ServiceName)

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

			lbConfigMap[service.ServiceName] = cfg
		}
	}

	// transform
	lbConfigs := make([]model.LBConfig, len(lbConfigMap))

	for _, cfg := range lbConfigMap {
		lbConfigs = append(lbConfigs, cfg)
	}

	return lbConfigs, nil
}
