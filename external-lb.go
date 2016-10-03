package main

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/rancher/external-lb/model"
	"strings"
)

type Op int

const (
	ADD Op = iota
	REMOVE
	UPDATE
)

func UpdateProviderLBConfigs(metadataConfigs map[string]model.LBConfig) (map[string]model.LBConfig, error) {
	providerConfigs, err := getProviderLBConfigs()
	if err != nil {
		return nil, fmt.Errorf("Failed to get LB configs from provider: %v", err)
	}

	removeExtraConfigs(metadataConfigs, providerConfigs)
	updated := addMissingConfigs(metadataConfigs, providerConfigs)
	updated_ := updateExistingConfigs(metadataConfigs, providerConfigs)
	for k, v := range updated_ {
		if _, ok := updated[k]; !ok {
			updated[k] = v
		}
	}

	return updated, nil
}

func getProviderLBConfigs() (map[string]model.LBConfig, error) {
	allConfigs, err := provider.GetLBConfigs()
	if err != nil {
		return nil, err
	}

	rancherConfigs := make(map[string]model.LBConfig, len(allConfigs))
	suffix := "_" + m.EnvironmentUUID + "_" + targetPoolSuffix
	for _, value := range allConfigs {
		if strings.HasSuffix(value.LBTargetPoolName, suffix) {
			rancherConfigs[value.LBEndpoint] = value
		}
	}

	logrus.Debugf("LBConfigs from provider: %v", allConfigs)
	return rancherConfigs, nil
}

func removeExtraConfigs(metadataConfigs, providerConfigs map[string]model.LBConfig) map[string]model.LBConfig {
	var toRemove []model.LBConfig
	for key := range providerConfigs {
		if _, ok := metadataConfigs[key]; !ok {
			toRemove = append(toRemove, providerConfigs[key])
		}
	}

	if len(toRemove) == 0 {
		logrus.Debug("No LB configs to remove")
	} else {
		logrus.Infof("LB configs to remove: %d", len(toRemove))
	}

	return updateProvider(toRemove, REMOVE)
}

func addMissingConfigs(metadataConfigs, providerConfigs map[string]model.LBConfig) map[string]model.LBConfig {
	var toAdd []model.LBConfig
	for key := range metadataConfigs {
		if _, ok := providerConfigs[key]; !ok {
			toAdd = append(toAdd, metadataConfigs[key])
		}
	}

	if len(toAdd) == 0 {
		logrus.Debug("No LB configs to add")
	} else {
		logrus.Infof("LB configs to add: %d", len(toAdd))
	}

	return updateProvider(toAdd, ADD)
}

func updateExistingConfigs(metadataConfigs, providerConfigs map[string]model.LBConfig) map[string]model.LBConfig {
	var toUpdate []model.LBConfig
	for key := range metadataConfigs {
		if _, ok := providerConfigs[key]; ok {
			mLBConfig := metadataConfigs[key]
			pLBConfig := providerConfigs[key]
			var update bool

			//check that the targetPoolName and targets match
			if strings.EqualFold(mLBConfig.LBTargetPoolName, pLBConfig.LBTargetPoolName) {
				if len(mLBConfig.LBTargets) != len(pLBConfig.LBTargets) {
					update = true
				} else {
					//check if any target have changed
					for _, mTarget := range mLBConfig.LBTargets {
						targetExists := false
						for _, pTarget := range pLBConfig.LBTargets {
							if pTarget.HostIP == mTarget.HostIP && pTarget.Port == mTarget.Port {
								targetExists = true
								break
							}
						}
						if !targetExists {
							//lb target changed, update the config on provider
							update = true
							break
						}
					}
				}
			} else {
				//targetPool should be changed
				logrus.Debugf("The LBEndPoint %s  will be updated to map to a new LBTargetPoolName %s", key, mLBConfig.LBTargetPoolName)
				update = true
			}

			if update {
				toUpdate = append(toUpdate, metadataConfigs[key])
			}
		}
	}

	if len(toUpdate) == 0 {
		logrus.Debug("No LB configs to update")
	} else {
		logrus.Infof("LB configs to update: %d", len(toUpdate))
	}

	return updateProvider(toUpdate, UPDATE)
}

func updateProvider(toChange []model.LBConfig, op Op) map[string]model.LBConfig {
	// map of FQDN -> LBConfig
	updateFqdn := make(map[string]model.LBConfig)
	for _, value := range toChange {
		switch op {
		case ADD:
			logrus.Infof("Adding LB config: %v", value)
			fqdn, err := provider.AddLBConfig(value)
			if err != nil {
				logrus.Errorf("Failed to add LB config for endpoint %s: %v", value.LBEndpoint, err)
			} else if fqdn != "" {
				updateFqdn[fqdn] = value
			}
		case REMOVE:
			logrus.Infof("Removing LB config: %v", value)
			if err := provider.RemoveLBConfig(value); err != nil {
				logrus.Errorf("Failed to remove LB config for endpoint %s: %v", value.LBEndpoint, err)
			}
		case UPDATE:
			logrus.Infof("Updating LB config: %v", value)
			fqdn, err := provider.UpdateLBConfig(value)
			if err != nil {
				logrus.Errorf("Failed to update LB config for endpoint %s: %v", value.LBEndpoint, err)
			} else if fqdn != "" {
				updateFqdn[fqdn] = value
			}
		}
	}

	return updateFqdn
}
