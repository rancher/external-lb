package avi

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/external-lb/model"
	"github.com/rancher/external-lb/providers"
)

const (
	ProviderName = "Avi"

	AVI_USER            = "AVI_USER"
	AVI_PASSWORD        = "AVI_PASSWORD"
	AVI_CONTROLLER_ADDR = "AVI_CONTROLLER_ADDR"
	AVI_CONTROLLER_PORT = "AVI_CONTROLLER_PORT"
	AVI_SSL_VERIFY      = "AVI_SSL_VERIFY"
	AVI_CA_CERT_PATH    = "AVI_CA_CERT_PATH"

	// Assume VSes already configured, so following not needed
	// AVI_CLOUD_NAME      = "AVI_CLOUD_NAME"
	// AVI_DNS_SUBDOMAIN   = "AVI_DNS_SUBDOMAIN"
)

type AviConfig struct {
	controllerIpAddr string
	controllerPort   int
	username         string
	password         string
	sslVerify        bool
	caCertPath       string

	cloudName    string
	dnsSubDomain string
}

type AviProvider struct {
	aviSession *AviSession
	cfg        *AviConfig
}

func log() *logrus.Entry {
	return logrus.WithFields(logrus.Fields{
		"provider": ProviderName,
	})
}

func init() {
	providers.RegisterProvider(ProviderName, new(AviProvider))
}

func getAviConfig() (*AviConfig, error) {
	conf := make(map[string]string)
	conf[AVI_USER] = os.Getenv(AVI_USER)
	conf[AVI_PASSWORD] = os.Getenv(AVI_PASSWORD)
	conf[AVI_CONTROLLER_ADDR] = os.Getenv(AVI_CONTROLLER_ADDR)
	conf[AVI_CONTROLLER_PORT] = os.Getenv(AVI_CONTROLLER_PORT)
	conf[AVI_SSL_VERIFY] = os.Getenv(AVI_SSL_VERIFY)
	conf[AVI_CA_CERT_PATH] = os.Getenv(AVI_CA_CERT_PATH)

	// Assume VSes already configured, so following not needed
	// conf[AVI_CLOUD_NAME] = os.Getenv(AVI_CLOUD_NAME)
	// conf[AVI_DNS_SUBDOMAIN] = os.Getenv(AVI_DNS_SUBDOMAIN)

	b, _ := json.MarshalIndent(conf, "", " ")
	log().Infof("Configured provider %s with values %s \n",
		ProviderName, string(b))

	return validateConfig(conf)
}

func validateConfig(conf map[string]string) (*AviConfig, error) {
	cfg := new(AviConfig)
	if conf[AVI_USER] == "" {
		return cfg, fmt.Errorf("AVI_USER not set")
	}
	cfg.username = conf[AVI_USER]

	if conf[AVI_PASSWORD] == "" {
		return cfg, fmt.Errorf("AVI_PASSWORD not set")
	}
	cfg.password = conf[AVI_PASSWORD]

	if conf[AVI_CONTROLLER_ADDR] == "" {
		return cfg, fmt.Errorf("AVI_CONTROLLER_ADDR not set")
	}
	cfg.controllerIpAddr = conf[AVI_CONTROLLER_ADDR]

	var port int
	var err error
	if conf[AVI_CONTROLLER_PORT] == "" {
		log().Warn("AVI_CONTROLLER_PORT is not set, using 443 as default.")
		port = 443
	} else {
		port, err = strconv.Atoi(conf[AVI_CONTROLLER_PORT])
		if err != nil {
			log().Errorf("Error controller port: %v", err)
			return cfg, err
		}
	}
	cfg.controllerPort = port

	sslVerify := true
	if conf[AVI_SSL_VERIFY] == "no" || conf[AVI_SSL_VERIFY] == "false" {
		conf[AVI_SSL_VERIFY] = "false"
		sslVerify = false
	} else {
		conf[AVI_SSL_VERIFY] = "true"
	}

	if sslVerify {
		if conf[AVI_CA_CERT_PATH] != "" {
			// check if path exists
			log().Info("Using system default path for CA certificates")
		} else {
			log().Infof("Using CA certificate path %s", conf[AVI_CA_CERT_PATH])
		}
	}
	cfg.sslVerify = sslVerify
	cfg.caCertPath = conf[AVI_CA_CERT_PATH]

	//if conf[AVI_CLOUD_NAME] == "" {
	// log().Info("AVI_CLOUD_NAME not set, using Default-Cloud")
	// conf[AVI_CLOUD_NAME] = "Default-Cloud"
	//}

	//if conf[AVI_DNS_SUBDOMAIN] == "" {
	//log().Info("AVI_DNS_SUBDOMAIN not set")
	//}

	cfg.cloudName = "Default-Cloud"
	cfg.dnsSubDomain = ""

	return cfg, nil
}

func initAviSession(cfg *AviConfig) (*AviSession, error) {
	insecure := !cfg.sslVerify
	portStr := strconv.Itoa(cfg.controllerPort)
	netloc := cfg.controllerIpAddr + ":" + portStr // 10.0.1.4:9443 typish
	aviSession := NewAviSession(netloc,
		cfg.username,
		cfg.password,
		insecure)
	err := aviSession.InitiateSession()
	return aviSession, err
}

func (p *AviProvider) Init() error {
	cfg, err := getAviConfig()
	if err != nil {
		return err
	}

	p.cfg = cfg
	aviSession, err := initAviSession(cfg)
	if err != nil {
		return err
	}

	p.aviSession = aviSession
	log().Info("Avi configuration OK")
	return nil
}

func (p *AviProvider) GetName() string {
	return ProviderName
}

func (p *AviProvider) HealthCheck() error {
	log().Info("Running health check on Avi...")
	_, err := p.aviSession.Get("")
	if err != nil {
		log().Errorf("Health check failed with error %s", err)
		return err
	}

	log().Info("Connection to Avi is OK")
	return nil
}

func (p *AviProvider) updateVs(vs AviRawDataType) error {
	vsUuid := vs["uuid"].(string)
	uri := "/api/virtualservice?uuid=" + vsUuid
	_, err := p.aviSession.Put(uri, vs)
	return err
}

func (p *AviProvider) checkExisitngPool(vs AviRawDataType, poolName string) (AviRawDataType, error) {
	empty := make(AviRawDataType)
	poolUrl := vs["pool_ref"].(string)
	u, err := url.Parse(poolUrl)
	if err != nil {
		return empty, fmt.Errorf("Invlid pool ref [%s]", poolUrl)
	}

	aviPool, err := p.GetPool(u.Path)
	if err != nil {
		return empty, err
	}

	aviPoolName := aviPool["name"].(string)
	if aviPoolName != poolName {
		aviPool["name"] = poolName
	}

	return aviPool, nil
}

func (p *AviProvider) ensureVsHasPool(vs AviRawDataType, poolName string) (AviRawDataType, error) {
	empty := make(AviRawDataType)
	if _, ok := vs["pool_ref"]; !ok {
		// pool doesn't exist; create one
		pool, err := p.EnsurePoolExists(poolName)
		if err != nil {
			return empty, err
		}
		vs["pool_ref"] = pool["url"]
		err = p.updateVs(vs)
		if err != nil {
			return empty, err
		}
		return pool, nil
	}

	return p.checkExisitngPool(vs, poolName)
}

func (p *AviProvider) addNewMembersToPool(pool AviRawDataType, config model.LBConfig) error {
	vsName := config.LBEndpoint
	dockerTasks := NewDockerTasks()
	for _, host := range config.LBTargets {
		hostPort, _ := strconv.Atoi(host.Port)
		dt := NewDockerTask(vsName, "tcp", host.HostIP, hostPort, -1)
		dockerTasks[dt.Key()] = dt
	}

	err := p.AddPoolMembers(pool, dockerTasks)
	if err != nil {
		return err
	}

	return nil
}

func (p *AviProvider) AddLBConfig(config model.LBConfig) (string, error) {
	vsName := config.LBEndpoint
	vs, err := p.GetVS(vsName)
	if err != nil {
		return "", err
	}

	poolName := config.LBTargetPoolName
	pool, err := p.ensureVsHasPool(vs, poolName)
	if err != nil {
		return "", err
	}

	err = p.addNewMembersToPool(pool, config)
	if err != nil {
		return "", nil
	}

	fqdn, err := GetVsFqdn(vs)
	if err != nil {
		log().Warnf("%s", err)
	}
	return fqdn, nil
}

func (p *AviProvider) RemoveLBConfig(config model.LBConfig) error {
	//TODO
	log().Infof("RemoveLBConfig called with config %s", config)
	return nil
}

func (p *AviProvider) UpdateLBConfig(config model.LBConfig) (string, error) {
	//TODO
	log().Infof("UpdateLBConfig called with config %s", config)
	return "", nil
}

func formLBConfig(vs AviRawDataType, pool AviRawDataType) model.LBConfig {
	lbTargets := make([]model.LBTarget, 0)
	targetPort := ""
	currMembers := getPoolMembers(pool)
	for _, server := range currMembers {
		ip := server["ip"].(AviRawDataType)
		ipAddr := ip["addr"].(string)
		port := strconv.FormatInt(int64(server["port"].(float64)), 10)
		lbTargets = append(lbTargets, model.LBTarget{ipAddr, port})
		targetPort = port
	}

	vsName := vs["name"].(string)
	poolName := pool["name"].(string)
	return model.LBConfig{vsName, poolName, targetPort, lbTargets}
}

func (p *AviProvider) GetLBConfigs() ([]model.LBConfig, error) {
	log().Infof("Fetching virtual services from Avi...")
	lbConfigs := make([]model.LBConfig, 0)
	allVses, err := p.GetAllVses()
	if err != nil {
		return lbConfigs, err
	}

	for _, vs := range allVses {
		poolUrl := vs["pool_ref"].(string)
		u, _ := url.Parse(poolUrl)
		pool, err := p.GetPool(u.Path)
		if err != nil {
			return lbConfigs, err
		}

		lbConfig := formLBConfig(vs, pool)
		lbConfigs = append(lbConfigs, lbConfig)
	}

	log().Infof("=== GetLBConfig value %s", lbConfigs)
	return lbConfigs, nil
}
