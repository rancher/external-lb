package main

import (
	"flag"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/external-lb/metadata"
	"github.com/rancher/external-lb/model"
	"github.com/rancher/external-lb/providers"
	_ "github.com/rancher/external-lb/providers/aliyunslb"
	_ "github.com/rancher/external-lb/providers/avi"
	_ "github.com/rancher/external-lb/providers/elbv1"
	_ "github.com/rancher/external-lb/providers/f5"
)

const (
	pollInterval = 1000
	// if metadata wasn't updated in 1 min, force update would be executed
	forceUpdateInterval = 1
)

var (
	providerName    = flag.String("provider", "f5_BigIP", "External LB  provider name")
	debug           = flag.Bool("debug", false, "Debug")
	logFile         = flag.String("log", "", "Log file")
	metadataAddress = flag.String("metadata-address", "rancher-metadata", "The metadata service address")

	provider providers.Provider
	m        *metadata.MetadataClient
	c        *CattleClient

	targetPoolSuffix        string
	metadataLBConfigsCached = make(map[string]model.LBConfig)
)

func setEnv() {
	flag.Parse()
	if *debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if *logFile != "" {
		if output, err := os.OpenFile(*logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666); err != nil {
			logrus.Fatalf("Failed to log to file %s: %v", *logFile, err)
		} else {
			logrus.SetOutput(output)
			formatter := &logrus.TextFormatter{
				FullTimestamp: true,
			}
			logrus.SetFormatter(formatter)
		}
	}

	// initialize metadata client
	var err error
	m, err = metadata.NewMetadataClient(*metadataAddress)
	if err != nil {
		logrus.Fatalf("Failed to initialize Rancher metadata client: %v", err)
	}

	// initialize cattle client
	c, err = NewCattleClientFromEnvironment()
	if err != nil {
		logrus.Fatalf("Failed to initialize Rancher API client: %v", err)
	}

	// initialize provider
	provider, err = providers.GetProvider(*providerName)
	if err != nil {
		logrus.Fatalf("Failed to initialize provider '%s': %v", *providerName, err)
	}

	targetPoolSuffix = os.Getenv("LB_TARGET_RANCHER_SUFFIX")
	if len(targetPoolSuffix) == 0 {
		logrus.Info("LB_TARGET_RANCHER_SUFFIX is not set, using default suffix 'rancher.internal'")
		targetPoolSuffix = "rancher.internal"
	}

}

func main() {
	logrus.Infof("Starting Rancher External LoadBalancer service")
	setEnv()

	go startHealthcheck()

	version := "init"
	lastUpdated := time.Now()

	ticker := time.NewTicker(time.Duration(pollInterval) * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		update, updateForced := false, false
		newVersion, err := m.GetVersion()
		if err != nil {
			logrus.Errorf("Failed to get metadata version: %v", err)
		} else if version != newVersion {
			logrus.Debugf("Metadata version changed. Old: %s New: %s.", version, newVersion)
			version = newVersion
			update = true
		} else {
			if time.Since(lastUpdated).Minutes() >= forceUpdateInterval {
				logrus.Debugf("Executing force update as metadata version hasn't changed in: %d minutes",
					forceUpdateInterval)
				updateForced = true
			}
		}

		if update || updateForced {
			// get records from metadata
			metadataLBConfigs, err := m.GetMetadataLBConfigs(targetPoolSuffix)
			if err != nil {
				logrus.Errorf("Failed to get LB configs from metadata: %v", err)
				continue
			}

			logrus.Debugf("LB configs from metadata: %v", metadataLBConfigs)

			// A flapping service might cause the metadata version to change
			// in short intervals. Caching the previous LB Configs allows
			// us to check if the actual LB Configs have changed, so we
			// don't end up flooding the provider with unnecessary requests.
			if !reflect.DeepEqual(metadataLBConfigs, metadataLBConfigsCached) || updateForced {
				// update the provider
				updatedFqdn, err := UpdateProviderLBConfigs(metadataLBConfigs)
				if err != nil {
					logrus.Errorf("Failed to update provider: %v", err)
				}

				// update the service FQDN in Cattle
				for fqdn, config := range updatedFqdn {
					// service_stack_environment_suffix
					parts := strings.Split(config.LBTargetPoolName, "_")
					err := c.UpdateServiceFqdn(parts[0], parts[1], fqdn)
					if err != nil {
						logrus.Errorf("Failed to update service FQDN: %v", err)
					}
				}

				metadataLBConfigsCached = metadataLBConfigs
				lastUpdated = time.Now()

			} else {
				logrus.Debugf("LB configs from metadata did not change")
			}
		}
	}
}
