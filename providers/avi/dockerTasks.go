package avi

import (
	"strconv"
	"strings"
)

// task maps to pool member in Avi
type dockerTask struct {
	serviceName string
	portType    string // tcp/udp
	ipAddr      string // host IP Address hosting container
	publicPort  int    // publicly exposed port
	privatePort int    // private port
}

func NewDockerTask(serviceName string,
	portType string,
	ipAddr string,
	publicPort int,
	privatePort int) *dockerTask {
	return &dockerTask{serviceName, portType, ipAddr, publicPort, privatePort}
}

func makeKey(ipAddr string, port string) string {
	const sep = "-"
	return strings.Join([]string{ipAddr, port}, sep)
}

func (dt *dockerTask) Key() string {
	port := strconv.Itoa(dt.publicPort)
	return makeKey(dt.ipAddr, port)
}

// taskKey -> dockerTask
type dockerTasks map[string]*dockerTask

func NewDockerTasks() dockerTasks {
	return make(dockerTasks)
}
