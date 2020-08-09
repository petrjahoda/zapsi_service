package main

import (
	"github.com/kardianos/service"
	"github.com/petrjahoda/database"
	"strconv"
	"sync"
	"time"
)

const version = "2020.3.2.9"
const serviceName = "Zapsi Service"
const serviceDescription = "Downloads data from Zapsi devices"
const downloadInSeconds = 10
const config = "user=postgres password=Zps05..... dbname=version3 host=database port=5432 sslmode=disable"

var serviceRunning = false
var serviceDirectory string

var (
	activeDevices  []database.Device
	runningDevices []database.Device
	deviceSync     sync.Mutex
)

type program struct{}

func main() {
	LogInfo("MAIN", serviceName+" ["+version+"] starting...")
	serviceConfig := &service.Config{
		Name:        serviceName,
		DisplayName: serviceName,
		Description: serviceDescription,
	}
	prg := &program{}
	s, err := service.New(prg, serviceConfig)
	if err != nil {
		LogError("MAIN", "Cannot start: "+err.Error())
	}
	err = s.Run()
	if err != nil {
		LogError("MAIN", "Cannot start: "+err.Error())
	}
}
func (p *program) Start(service.Service) error {
	LogInfo("MAIN", serviceName+" ["+version+"] started")
	go p.run()
	serviceRunning = true
	return nil
}

func (p *program) Stop(service.Service) error {
	serviceRunning = false
	for len(runningDevices) != 0 {
		LogInfo("MAIN", serviceName+" ["+version+"] stopping...")
		time.Sleep(1 * time.Second)
	}
	LogInfo("MAIN", serviceName+" ["+version+"] stopped")
	return nil
}

func (p *program) run() {
	UpdateProgramVersion()
	for {
		LogInfo("MAIN", serviceName+" ["+version+"] running")
		start := time.Now()
		ReadActiveDevices("MAIN")
		LogInfo("MAIN", "Active devices: "+strconv.Itoa(len(activeDevices))+", running devices: "+strconv.Itoa(len(runningDevices)))
		for _, activeDevice := range activeDevices {
			activeDeviceIsRunning := CheckDeviceInRunningDevices(activeDevice)
			if !activeDeviceIsRunning {
				go RunDevice(activeDevice)
			}
		}
		if time.Since(start) < (downloadInSeconds * time.Second) {
			sleepTime := downloadInSeconds*time.Second - time.Since(start)
			LogInfo("MAIN", "Sleeping for "+sleepTime.String())
			time.Sleep(sleepTime)
		}
	}
}
