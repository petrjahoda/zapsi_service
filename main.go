package main

import (
	"github.com/jinzhu/gorm"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const version = "2019.4.1.29"
const deleteLogsAfter = 240 * time.Hour
const downloadInSeconds = 10

var (
	activeDevices  []Device
	runningDevices []Device
	deviceSync     sync.Mutex
)

func main() {
	LogDirectoryFileCheck("MAIN")
	LogInfo("MAIN", "Program version "+version+" started")
	CreateConfigIfNotExists()
	LoadSettingsFromConfigFile()
	LogDebug("MAIN", "Using ["+DatabaseType+"] on "+DatabaseIpAddress+":"+DatabasePort+" with database "+DatabaseName)
	SendMail("Program started", "Zapsi Service version "+version+" started")
	for {
		start := time.Now()
		LogInfo("MAIN", "Program running")
		CheckDatabase()
		CheckTables()
		UpdateActiveDevices("MAIN")
		DeleteOldLogFiles()
		LogInfo("MAIN", "Active devices: "+strconv.Itoa(len(activeDevices))+", running devices: "+strconv.Itoa(len(runningDevices)))
		for _, activeDevice := range activeDevices {
			activeDeviceIsRunning := CheckDevice(activeDevice)
			if !activeDeviceIsRunning {
				go RunDevice(activeDevice)
			}
		}
		if time.Since(start) < (downloadInSeconds * time.Second) {
			sleeptime := downloadInSeconds*time.Second - time.Since(start)
			LogInfo("MAIN", "Sleeping for "+sleeptime.String())
			time.Sleep(sleeptime)
		}
	}
}

func CheckDevice(device Device) bool {
	for _, runningDevice := range runningDevices {
		if runningDevice.Name == device.Name {
			return true
		}
	}
	return false
}

func RunDevice(device Device) {
	LogInfo(device.Name, "Device started running")
	deviceSync.Lock()
	runningDevices = append(runningDevices, device)
	deviceSync.Unlock()
	deviceIsActive := true
	device.CreateDirectoryIfNotExists()
	device.SendTime()
	timeUpdatedInLoop := false
	for deviceIsActive {
		start := time.Now()
		ProcessDownloadedFiles(device)
		success, err := device.DownloadData()
		if err != nil {
			LogError(device.Name, "Error downloading data: "+err.Error())
		}
		if success {
			ProcessDownloadedFiles(device)
		}
		LogInfo(device.Name, "Processing takes "+time.Since(start).String())
		timeUpdatedInLoop = device.SendTimeToZapsi(timeUpdatedInLoop)
		device.Sleep(start)
		deviceIsActive = CheckActive(device)
	}
	RemoveDeviceFromRunningDevice(device)
	LogInfo(device.Name, "Device not active, stopped running")

}

func ProcessDownloadedFiles(device Device) {
	intermediateData := device.PrepareData()
	if len(intermediateData) > 0 {
		err := device.ProcessData(intermediateData)
		if err != nil {
			LogError(device.Name, "Error processing data: "+err.Error())
		}
	}
	DeleteDownloadedFile("digital.txt", device)
	DeleteDownloadedFile("analog.txt", device)
	DeleteDownloadedFile("serial.txt", device)
	DeleteDownloadedFile("ui_value.txt", device)
}

func DeleteDownloadedFile(deviceFileName string, device Device) {
	deviceDirectory := filepath.Join(".", strconv.Itoa(int(device.ID))+"-"+device.Name)
	deviceFullPath := strings.Join([]string{deviceDirectory, deviceFileName}, "/")
	info, err := os.Stat(deviceFullPath)
	if err != nil {
		LogDebug(device.Name, "File does not exist: "+err.Error())
		return
	}
	if !info.IsDir() {
		err := os.Remove(deviceFullPath)
		if err != nil {
			LogError(device.Name, "Problem deleting file, "+err.Error())
		}
	}
}

func CheckActive(device Device) bool {
	for _, activeDevice := range activeDevices {
		if activeDevice.Name == device.Name {
			LogInfo(device.Name, "Device still active")
			return true
		}
	}
	LogInfo(device.Name, "Device not active")
	return false
}

func RemoveDeviceFromRunningDevice(device Device) {
	for idx, runningDevice := range runningDevices {
		if device.Name == runningDevice.Name {
			deviceSync.Lock()
			runningDevices = append(runningDevices[0:idx], runningDevices[idx+1:]...)
			deviceSync.Unlock()
		}
	}
}

func UpdateActiveDevices(reference string) {
	connectionString, dialect := CheckDatabaseType()
	db, err := gorm.Open(dialect, connectionString)
	if err != nil {
		LogError(reference, "Problem opening "+DatabaseName+" database: "+err.Error())
		return
	}
	defer db.Close()
	var deviceType DeviceType
	db.Where("name=?", "Zapsi").Find(&deviceType)
	db.Where("device_type_id=?", deviceType.ID).Where("is_activated = true").Find(&activeDevices)
	LogDebug("MAIN", "Zapsi device type id is "+strconv.Itoa(int(deviceType.ID)))
}
