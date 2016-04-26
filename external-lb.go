package main

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/rancher/external-lb/model"
	"strings"
)

func UpdateProviderLBConfigs(metadataConfigs map[string]model.LBConfig) error {
	providerConfigs, err := getProviderLBConfigs()
	if err != nil {
		return fmt.Errorf("Provider error reading lb configs: %v", err)
	}
	logrus.Debugf("Rancher LB configs from provider: %v", providerConfigs)

	removeExtraConfigs(metadataConfigs, providerConfigs)

	addMissingConfigs(metadataConfigs, providerConfigs)

	updateExistingConfigs(metadataConfigs, providerConfigs)

	return nil
}

func getProviderLBConfigs() (map[string]model.LBConfig, error) {
	allConfigs, err := provider.GetLBConfigs()
	if err != nil {
		logrus.Debugf("Error Getting Rancher LB configs from provider: %v", err)
		return nil, err
	}
	rancherConfigs := make(map[string]model.LBConfig, len(allConfigs))
	suffix := "_" + m.EnvironmentUUID + "_" + targetRancherSuffix
	for _, value := range allConfigs {
		if strings.HasSuffix(value.LBTargetPoolName, suffix) {
			rancherConfigs[value.LBEndpoint] = value
		}
	}
	return rancherConfigs, nil
}

func removeExtraConfigs(metadataConfigs map[string]model.LBConfig, providerConfigs map[string]model.LBConfig) []model.LBConfig {
	var toRemove []model.LBConfig
	for key := range providerConfigs {
		if _, ok := metadataConfigs[key]; !ok {
			toRemove = append(toRemove, providerConfigs[key])
		}
	}

	if len(toRemove) == 0 {
		logrus.Debug("No LB configs to remove")
	} else {
		logrus.Infof("LB configs to remove: %v", toRemove)
	}
	return updateProvider(toRemove, &Remove)
}

func addMissingConfigs(metadataConfigs map[string]model.LBConfig, providerConfigs map[string]model.LBConfig) []model.LBConfig {
	var toAdd []model.LBConfig
	for key := range metadataConfigs {
		if _, ok := providerConfigs[key]; !ok {
			toAdd = append(toAdd, metadataConfigs[key])
		}
	}
	if len(toAdd) == 0 {
		logrus.Debug("No LB configs to add")
	} else {
		logrus.Infof("LB configs to add: %v", toAdd)
	}
	return updateProvider(toAdd, &Add)
}

func updateExistingConfigs(metadataConfigs map[string]model.LBConfig, providerConfigs map[string]model.LBConfig) []model.LBConfig {
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
				}
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
		logrus.Infof("LB configs to update: %v", toUpdate)
	}

	return updateProvider(toUpdate, &Update)
}

func updateProvider(toChange []model.LBConfig, op *Op) []model.LBConfig {
	var changed []model.LBConfig
	for _, value := range toChange {
		switch *op {
		case Add:
			logrus.Infof("Adding LB config: %v", value)
			if err := provider.AddLBConfig(value); err != nil {
				logrus.Errorf("Failed to add LB config to provider %v: %v", value, err)
			} else {
				changed = append(changed, value)
			}
		case Remove:
			logrus.Infof("Removing LB config: %v", value)
			if err := provider.RemoveLBConfig(value); err != nil {
				logrus.Errorf("Failed to remove LB config from provider %v: %v", value, err)
			}
		case Update:
			logrus.Infof("Updating LB config: %v", value)
			if err := provider.UpdateLBConfig(value); err != nil {
				logrus.Errorf("Failed to update LB config to provider %v: %v", value, err)
			} else {
				changed = append(changed, value)
			}
		}
	}
	return changed
}
