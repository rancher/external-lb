package model

type LBConfig struct {
	LBEndpoint   string
	LBTargetPoolName string
	LBTargets    []LBTarget
}

type LBTarget struct {
	HostIP string
	Port   string
}
