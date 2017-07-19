package model

import (
	"fmt"
)

type LBConfig struct {
	LBEndpoint       string
	LBTargetPoolName string
	LBTargetPort     string
	LBTargets        []LBTarget
}

type LBTarget struct {
	HostIP string
	Port   string
}

func (t LBTarget) String() string {
	return fmt.Sprintf("%s:%s", t.HostIP, t.Port)
}

func (c LBConfig) String() string {
	lbTargets := ""
	for _, t := range c.LBTargets {
		s := "(" + fmt.Sprintf("%s", t) + ") "
		lbTargets += s
	}
	return fmt.Sprintf("LBConfig [Endpoint: %s, Pool Name: %s, TargetPort: %s, Targets: %s]",
		c.LBEndpoint, c.LBTargetPoolName, c.LBTargetPort, lbTargets)
}
