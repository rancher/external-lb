package graviteelb

import (
	"crypto/sha1"
	"fmt"
	"io"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/konsorten/go-gravitee"
	"github.com/rancher/external-lb/model"
	"github.com/rancher/external-lb/providers"
)

const (
	providerName = "Gravitee"
	providerSlug = "gravitee"
)

type GraviteeProvider struct {
	client      *gravitee.GraviteeSession
	configCache map[string]string
}

func init() {
	providers.RegisterProvider(providerSlug, new(GraviteeProvider))
}

func getApiMetadataMap(md []gravitee.ApiMetadata) map[string]string {
	r := make(map[string]string)

	for _, m := range md {
		r[m.Key] = m.Value()
	}

	return r
}

func getConfigHash(config *model.LBConfig) string {
	h := sha1.New()
	io.WriteString(h, config.LBEndpoint)
	io.WriteString(h, "###")

	labels := make([]string, 0)

	for k, _ := range config.LBLabels {
		labels = append(labels, k)
	}

	sort.Strings(labels)

	for _, k := range labels {
		v := config.LBLabels[k]

		io.WriteString(h, k)
		io.WriteString(h, ":::")
		io.WriteString(h, v)
		io.WriteString(h, ";;;")
	}

	io.WriteString(h, "###")
	io.WriteString(h, config.LBTargetPoolName)
	io.WriteString(h, "###")
	io.WriteString(h, config.LBTargetPort)
	io.WriteString(h, "###")

	for _, v := range config.LBTargets {
		io.WriteString(h, v.HostIP)
		io.WriteString(h, ":")
		io.WriteString(h, v.Port)
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}

func (p *GraviteeProvider) Init() (err error) {
	host := os.Getenv("GRAVITEE_HOST")
	if len(host) == 0 {
		return fmt.Errorf("GRAVITEE_HOST is not set")
	}

	user := os.Getenv("GRAVITEE_USER")
	if len(user) == 0 {
		return fmt.Errorf("GRAVITEE_USER is not set")
	}

	pwd := os.Getenv("GRAVITEE_PWD")
	if len(pwd) == 0 {
		return fmt.Errorf("GRAVITEE_PWD is not set")
	}

	log.Debugf("Initializing Gravitee provider: %s, user: %s, pwd-length: %d", host, user, len(pwd))

	p.client, err = gravitee.Connect(host, user, pwd, nil)

	if err != nil {
		return
	}

	p.configCache = make(map[string]string)

	log.Infof("Configured %s provider using %s", p.GetName(), host)
	return
}

func (p *GraviteeProvider) GetName() string {
	return providerName
}

func (p *GraviteeProvider) HealthCheck() error {
	success, msg := p.client.Ping()

	if !success {
		return fmt.Errorf("Failed to ping Gravitee loadbalancer: %v", msg)
	}

	return nil
}

func (p *GraviteeProvider) AddLBConfig(config model.LBConfig) (string, error) {
	// first check if changes can be made
	if available, msg := p.client.Ping(); !available {
		return "", fmt.Errorf("Failed to ping Gravitee loadbalancer: %v", msg)
	}

	apis, err := p.client.GetAPIsByLabel(config.LBEndpoint)
	if err != nil {
		log.Errorf("Failed to retrieve APIs with label %v: %v", config.LBEndpoint, err)
		return "", nil
	}
	if apis == nil || len(apis) <= 0 {
		log.Warnf("No APIs found with with label %v", config.LBEndpoint)
		return "", nil
	}

	// add configurations
	for _, api := range apis {
		// configure
		_, err := p.addLBConfigSingle(api, config)
		if err != nil {
			log.Errorf("Failed to update API %v: %v", api, err)
		}
	}

	return "", nil
}

func (p *GraviteeProvider) addLBConfigSingle(api gravitee.ApiInfo, config model.LBConfig) (string, error) {
	// check if the config did change
	configHash := getConfigHash(&config)

	if hash, ok := p.configCache[api.ID]; ok && hash == configHash {
		log.Infof("Skipping config update of unchanged API %v", api)
		return "", nil
	}

	log.Debugf("Adding endpoints on API: %v", api)
	log.Debugf("Service labels: %v", config.LBLabels)

	// parse labels
	encryptedBackendsStr, _ := config.LBLabels["encrypt"]
	encryptedBackends := encryptedBackendsStr == "true"

	keepAliveStr, _ := config.LBLabels["keep_alive"]
	keepAlive := keepAliveStr == "true" || keepAliveStr == ""

	pipeliningStr, _ := config.LBLabels["pipelining"]
	pipelining := pipeliningStr == "true"

	compressStr, _ := config.LBLabels["compress"]
	compress := compressStr == "true" || compressStr == ""

	followRedirectsStr, _ := config.LBLabels["follow_redirects"]
	followRedirects := followRedirectsStr == "true"

	connectTimeoutStr, _ := config.LBLabels["conn_timeout"]
	readTimeoutStr, _ := config.LBLabels["read_timeout"]
	idleTimeoutStr, _ := config.LBLabels["idle_timeout"]
	maxConnectionsStr, _ := config.LBLabels["max_conn"]

	// update endpoints
	endpoints := make([]gravitee.ApiDetailsEndpoint, 0)

	for _, ep := range config.LBTargets {
		e := gravitee.MakeApiDetailsEndpoint(ep.HostIP, fmt.Sprintf("http://%v:%v", ep.HostIP, ep.Port))

		e.Http.KeepAlive = keepAlive
		e.Http.Pipelining = pipelining
		e.Http.UseCompression = compress
		e.Http.FollowRedirects = followRedirects
		e.SSL.IsEnabled = encryptedBackends

		if connectTimeoutStr != "" {
			e.Http.ConnectTimeoutMS, _ = strconv.Atoi(connectTimeoutStr)
		}
		if readTimeoutStr != "" {
			e.Http.ReadTimeoutMS, _ = strconv.Atoi(readTimeoutStr)
		}
		if idleTimeoutStr != "" {
			e.Http.IdleTimeoutMS, _ = strconv.Atoi(idleTimeoutStr)
		}
		if maxConnectionsStr != "" {
			e.Http.MaxConcurrentConnections, _ = strconv.Atoi(maxConnectionsStr)
		}

		endpoints = append(endpoints, e)
	}

	err := p.client.AddOrUpdateEndpoints(api.ID, endpoints, true)
	if err != nil {
		return "", fmt.Errorf("Failed to update endpoints for API %v on Gravitee loadbalancer: %v", api, err)
	}

	// add meta-data
	p.client.SetLocalAPIMetadata(api.ID, "rancher-lb-pool-name", config.LBTargetPoolName, gravitee.ApiMetadataFormat_String)
	p.client.SetLocalAPIMetadata(api.ID, "rancher-lb-port", config.LBTargetPort, gravitee.ApiMetadataFormat_String)
	p.client.SetLocalAPIMetadata(api.ID, "rancher-lb-endpoint", config.LBEndpoint, gravitee.ApiMetadataFormat_String)
	p.client.SetLocalAPIMetadata(api.ID, "rancher-lb-hash", configHash, gravitee.ApiMetadataFormat_String)

	// deploy new api config
	log.Debugf("Deploying API: %v", api)

	err = p.client.DeployAPI(api.ID)

	if err != nil {
		return "", fmt.Errorf("Failed to deploy API %v on Gravitee loadbalancer: %v", api, err)
	}

	// update cache
	p.configCache[api.ID] = configHash

	return "", nil
}

func (p *GraviteeProvider) UpdateLBConfig(config model.LBConfig) (string, error) {
	return p.AddLBConfig(config)
}

func (p *GraviteeProvider) RemoveLBConfig(config model.LBConfig) error {
	// first check if changes can be made
	if available, msg := p.client.Ping(); !available {
		return fmt.Errorf("Failed to ping Gravitee loadbalancer: %v", msg)
	}

	apis, err := p.client.GetAPIsByLabel(config.LBEndpoint)
	if err != nil {
		log.Errorf("Failed to retrieve APIs with label %v: %v", config.LBEndpoint, err)
		return nil
	}
	if apis == nil || len(apis) <= 0 {
		log.Warnf("No APIs found with with label %v", config.LBEndpoint)
		return nil
	}

	// add configurations
	for _, api := range apis {
		// configure
		err := p.removeLBConfigSingle(api, config)
		if err != nil {
			log.Errorf("Failed to update API %v: %v", api, err)
		}
	}

	return nil
}

func (p *GraviteeProvider) removeLBConfigSingle(api gravitee.ApiInfo, config model.LBConfig) error {
	// remove from cache
	delete(p.configCache, api.ID)

	// clear all endpoints
	endpoints := make([]gravitee.ApiDetailsEndpoint, 0)

	err := p.client.AddOrUpdateEndpoints(api.ID, endpoints, true)
	if err != nil {
		return fmt.Errorf("Failed to update endpoints for API %v on Gravitee loadbalancer: %v", api, err)
	}

	// remove meta-data
	meta, err := p.client.GetAPIMetadata(api.ID)
	if err != nil {
		return fmt.Errorf("Failed to retrieve metadata for API %v on Gravitee loadbalancer: %v", api, err)
	}

	for _, m := range meta {
		if m.IsLocal() && strings.HasPrefix(m.Key, "rancher-lb-") {
			err := p.client.UnsetLocalAPIMetadata(api.ID, m.Key)
			if err != nil {
				log.Warnf("Failed to delete local metadata '%v' from API %v on Gravitee loadbalancer: %v", m.Key, api, err)
			}
		}
	}

	// deploy new api config
	log.Debugf("Deploying API: %v", api)

	err = p.client.DeployAPI(api.ID)
	if err != nil {
		return fmt.Errorf("Failed to deploy API %v on Gravitee loadbalancer: %v", api, err)
	}

	return nil
}

func (p *GraviteeProvider) GetLBConfigs() ([]model.LBConfig, error) {
	// first check if changes can be made
	if available, msg := p.client.Ping(); !available {
		return nil, fmt.Errorf("Failed to ping Gravitee loadbalancer: %v", msg)
	}

	// get all APIs
	apis, err := p.client.GetAllAPIs()
	if err != nil {
		return nil, fmt.Errorf("Failed to get APIs from Gravitee loadbalancer: %v", err)
	}

	lbConfigs := make([]model.LBConfig, 0)

	for _, apiInfo := range apis {
		// check if the farm exists
		log.Debugf("Gathering existing API: %v", apiInfo.Name)

		// get metadata (and check if this is a rancher API)
		metaRaw, err := p.client.GetAPIMetadata(apiInfo.ID)
		if err != nil {
			return nil, fmt.Errorf("Failed to get API metadata for %v from Gravitee loadbalancer: %v", apiInfo, err)
		}

		if true {
			found := false

			for _, m := range metaRaw {
				if m.IsLocal() && strings.HasPrefix(m.Key, "rancher-lb-") {
					found = true
					break
				}
			}

			if !found {
				// ignore this api
				continue
			}
		}

		meta := getApiMetadataMap(metaRaw)

		// retrieve details
		api, err := p.client.GetAPI(apiInfo.ID)
		if err != nil {
			return nil, fmt.Errorf("Failed to get API %v from Gravitee loadbalancer: %v", apiInfo, err)
		}
		if api == nil {
			return nil, fmt.Errorf("API not found on Gravitee loadbalancer: %v", apiInfo)
		}

		// build config
		cfg := model.LBConfig{}
		ok := false

		if cfg.LBTargetPoolName, ok = meta["rancher-lb-pool-name"]; !ok {
			log.Errorf("Failed to retrieve target pool name from 'rancher-lb-pool-name' API metadata for %v from Gravitee loadbalancer", apiInfo)
			continue
		}

		if cfg.LBTargetPort, ok = meta["rancher-lb-port"]; !ok {
			log.Debugf("Failed to retrieve virtual port from 'rancher-lb-port' API metadata for %v from Gravitee loadbalancer; Assuming port HTTPS (443) ...", apiInfo)

			cfg.LBTargetPort = "443"
		}

		if cfg.LBEndpoint, ok = meta["rancher-lb-endpoint"]; !ok {
			log.Errorf("Failed to retrieve target endpoint from 'rancher-lb-endpoint' API metadata for %v from Gravitee loadbalancer", apiInfo)
			continue
		}

		if configHash, ok := meta["rancher-lb-hash"]; ok {
			// add to cache, if new
			if _, kk := p.configCache[api.ID]; !kk {
				p.configCache[api.ID] = configHash
			}
		}

		// get endpoints
		for _, ep := range api.Proxy.Endpoints {
			log.Debugf("Found endpoint on API '%v': %v", api, ep.Name)

			epUrl, err := url.Parse(ep.Target)
			if err != nil {
				log.Warnf("Failed to parse target URL for %v endpoint on %v API from Gravitee loadbalancer: %v: %v", ep.Name, apiInfo, ep.Target, err)
				continue
			}

			cfg.LBTargets = append(cfg.LBTargets, model.LBTarget{
				HostIP: epUrl.Host,
				Port:   epUrl.Port(),
			})
		}

		// done
		lbConfigs = append(lbConfigs, cfg)
	}

	// transform
	return lbConfigs, nil
}
