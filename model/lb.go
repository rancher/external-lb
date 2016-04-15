package model

type LBConfig struct {
	LBEndpoint   string
	LBTargetName string
	LBTargets    []LBTarget
}

type LBTarget struct {
	HostIP string
	Port   string
}
