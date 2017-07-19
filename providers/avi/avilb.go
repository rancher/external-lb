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

func NewDockerTask(serviceName string, portType string, ipAddr string, publicPort int, privatePort int) *dockerTask {
	return &dockerTask{serviceName, portType, ipAddr, publicPort, privatePort}
}

func makeKey(ipAddr string, port string) string {
	sep := "-"
	return strings.Join([]string{ipAddr, port}, sep)
}

func (dt *dockerTask) Key() string {
	return makeKey(dt.ipAddr, strconv.Itoa(dt.publicPort))
}

// taskKey -> dockerTask
type dockerTasks map[string]*dockerTask

func NewDockerTasks() dockerTasks {
	return make(dockerTasks)
}

// serviceName -> (taskKey -> dockerTask)
type tasksCache map[string]dockerTasks

func NewTaskCache() tasksCache {
	return make(map[string]dockerTasks)
}

type currentConfig struct {
	services     map[string]bool // services added or deleted
	tasksAdded   tasksCache      // tasks added
	tasksDeleted tasksCache      // tasks deleted
}

func NewCurrentConfig() *currentConfig {
	services := make(map[string]bool)
	tasksAdded := NewTaskCache()
	tasksDeleted := NewTaskCache()

	return &currentConfig{services, tasksAdded, tasksDeleted}
}
