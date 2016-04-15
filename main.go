package main

import (
	"flag"
	"github.com/Sirupsen/logrus"
	"github.com/rancher/external-lb/metadata"
	"github.com/rancher/external-lb/providers"
	_ "github.com/rancher/external-lb/providers/f5"
	"os"
	"time"
)

const (
	poll = 1000
	// if metadata wasn't updated in 1 min, force update would be executed
	forceUpdateInterval = 1
)

type Op struct {
	Name string
}

var (
	Add    = Op{Name: "Add"}
	Remove = Op{Name: "Remove"}
	Update = Op{Name: "Update"}
)

var (
	providerName = flag.String("provider", "", "External LB  provider name")
	debug        = flag.Bool("debug", false, "Debug")
	logFile      = flag.String("log", "", "Log file")

	provider providers.Provider
	m        *metadata.MetadataClient
)

func setEnv() {
	flag.Parse()
	provider = providers.GetProvider(*providerName)
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

	// configure metadata client
	mClient, err := metadata.NewMetadataClient()
	if err != nil {
		logrus.Fatalf("Failed to configure rancher-metadata client: %v", err)
	}
	m = mClient

}

func main() {
	logrus.Infof("Starting Rancher External LoadBalancer service")
	setEnv()
	logrus.Infof("Powered by %s", provider.GetName())

	go startHealthcheck()

	version := "init"
	lastUpdated := time.Now()
	for {
		newVersion, err := m.GetVersion()
		update := false

		if err != nil {
			logrus.Errorf("Error reading metadata version: %v", err)
		} else if version != newVersion {
			logrus.Debugf("Metadata version has been changed. Old version: %s. New version: %s.", version, newVersion)
			version = newVersion
			update = true
		} else {
			logrus.Debugf("No changes in metadata version: %s", newVersion)
			if time.Since(lastUpdated).Minutes() >= forceUpdateInterval {
				logrus.Debugf("Executing force update as metadata version hasn't been changed in: %v minutes", forceUpdateInterval)
				update = true
			}
		}

		if update {
			// get records from metadata
			logrus.Debugf("Reading metadata LB Configs")
			metadataLBConfigs, err := m.GetMetadataLBConfigs()
			if err != nil {
				logrus.Errorf("Error reading metadata lb entries: %v", err)
			}
			logrus.Debugf("LB configs from metadata: %v", metadataLBConfigs)

			/*update provider*/
			logrus.Debugf("Reading Provider LB Configs")
			err = UpdateProviderLBConfigs(metadataLBConfigs)
			if err != nil {
				logrus.Errorf("Error reading provider lb entries: %v", err)
			}
			lastUpdated = time.Now()
		}

		time.Sleep(time.Duration(poll) * time.Millisecond)
	}
}
