package avi

import (
	"net/url"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/external-lb/model"
	"github.com/rancher/external-lb/providers"
)

var log *logrus.Entry

func initLogger() {
	log = logrus.WithFields(logrus.Fields{
		"provider": ProviderName,
	})
}

func init() {
	providers.RegisterProvider(ProviderName, new(AviProvider))
	initLogger()
}

type AviProvider struct {
	aviSession *AviSession
	cfg        *AviConfig
}

func (p *AviProvider) Init() error {
	cfg, err := GetAviConfig()
	if err != nil {
		return err
	}

	p.cfg = cfg
	aviSession, err := InitAviSession(cfg)
	if err != nil {
		return err
	}

	p.aviSession = aviSession
	log.Info("Avi configuration OK")
	return nil
}

func (p *AviProvider) GetName() string {
	return ProviderName
}

func (p *AviProvider) HealthCheck() error {
	log.Debug("Running health check on Avi...")
	_, err := p.aviSession.Get("")
	if err != nil {
		log.Errorf("Health check failed with error %s", err)
		return err
	}

	log.Debug("Connection to Avi is OK")
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
		log.Warnf("%s", err)
	}

	log.Debugf("FQDN for VS %s : %s", vsName, fqdn)
	return fqdn, nil
}

func (p *AviProvider) RemoveLBConfig(config model.LBConfig) error {
	vsName := config.LBEndpoint
	vs, err := p.GetVS(vsName)
	if err != nil {
		return err
	}

	if _, ok := vs["pool_ref"]; !ok {
		// pool doesn't exist; no op
		return nil
	}

	poolName := config.LBTargetPoolName
	pool, err := p.checkExisitngPool(vs, poolName)
	if err != nil {
		return err
	}

	err = p.removeMembersFromPool(pool, config)
	if err != nil {
		return err
	}

	return nil
}

func (p *AviProvider) UpdateLBConfig(config model.LBConfig) (string, error) {
	log.Infof("UpdateLBConfig called with config %s", config)
	return "", nil
}

func (p *AviProvider) GetLBConfigs() ([]model.LBConfig, error) {
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

	return lbConfigs, nil
}
