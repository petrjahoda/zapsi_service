package main

import (
	"github.com/petrjahoda/database"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func UpdateProgramVersion() {
	LogInfo("MAIN", "Writing program version into settings")
	timer := time.Now()
	db, err := gorm.Open(postgres.Open(config), &gorm.Config{})
	if err != nil {
		LogError("MAIN", "Problem opening database: "+err.Error())
		return
	}
	sqlDB, err := db.DB()
	defer sqlDB.Close()
	var existingSettings database.Setting
	db.Where("name=?", serviceName).Find(&existingSettings)
	existingSettings.Name = serviceName
	existingSettings.Value = version
	db.Save(&existingSettings)
	LogInfo("MAIN", "Program version written into settings in "+time.Since(timer).String())
}

func CheckDeviceInRunningDevices(device database.Device) bool {
	for _, runningDevice := range runningDevices {
		if runningDevice.Name == device.Name {
			return true
		}
	}
	return false
}

func RunDevice(device database.Device) {
	LogInfo(device.Name, "Device active, started running")
	deviceSync.Lock()
	runningDevices = append(runningDevices, device)
	deviceSync.Unlock()
	deviceIsActive := true
	CheckDeviceDataDirectory(device)
	SendTimeToDeviceAtStart(device)
	timeUpdatedInLoop := false
	for deviceIsActive && serviceRunning {
		LogInfo(device.Name, "Device main loop started")
		timer := time.Now()
		dataSuccessfullyProcessed := ProcessDownloadedData(device)
		if dataSuccessfullyProcessed {
			dataSuccessfullyDownloaded, err := DownloadDataFromDevice(device)
			if err != nil {
				LogError(device.Name, "Error downloading data: "+err.Error())
			}
			if dataSuccessfullyDownloaded {
				ProcessDownloadedData(device)
			}
		}
		timeUpdatedInLoop = SendTimeToDevice(device, timeUpdatedInLoop)
		LogInfo(device.Name, "Device main loop ended in "+time.Since(timer).String())
		Sleep(device, timer)
		deviceIsActive = CheckActive(device)
	}
	RemoveDeviceFromRunningDevices(device)
	LogInfo(device.Name, "Device not active, stopped running")

}

func CheckDeviceDataDirectory(device database.Device) {
	LogInfo(device.Name, "Checking device data directory")
	timer := time.Now()
	deviceDirectory := filepath.Join(serviceDirectory, strconv.FormatUint(uint64(device.ID), 10)+"-"+device.Name)
	if _, checkPathError := os.Stat(deviceDirectory); checkPathError == nil {
		LogInfo(device.Name, "Device directory already exists")
	} else if os.IsNotExist(checkPathError) {
		LogError(device.Name, "Device directory not exist, creating")
		mkdirError := os.MkdirAll(deviceDirectory, 0777)
		if mkdirError != nil {
			LogError(device.Name, "Unable to create device directory: "+mkdirError.Error())
		} else {
			LogInfo(device.Name, "Device directory created")
		}
	} else {
		LogError(device.Name, "Device directory does not exist")
	}
	LogInfo(device.Name, "Device data directory checked in "+time.Since(timer).String())
}

func Sleep(device database.Device, start time.Time) {
	if time.Since(start) < (downloadInSeconds * time.Second) {
		sleepTime := downloadInSeconds*time.Second - time.Since(start)
		LogInfo(device.Name, "Sleeping for "+sleepTime.String())
		time.Sleep(sleepTime)
	}
}

func ProcessDownloadedData(device database.Device) bool {
	LogInfo(device.Name, "Processing downloaded data")
	timer := time.Now()
	sortedData := PrepareDownloadedData(device)
	if len(sortedData) > 0 {
		err := ProcessSortedData(device, sortedData)
		if err != nil {
			LogError(device.Name, "Error processing data: "+err.Error())
			return false
		}
	}
	DeleteDownloadedData(device)
	LogInfo(device.Name, "Data processed in "+time.Since(timer).String())
	return true
}

func DeleteDownloadedData(device database.Device) {
	LogInfo(device.Name, "Deleting downloaded data")
	timer := time.Now()
	DeleteDownloadedFile("digital.txt", device)
	DeleteDownloadedFile("analog.txt", device)
	DeleteDownloadedFile("serial.txt", device)
	DeleteDownloadedFile("ui_value.txt", device)
	LogInfo(device.Name, "Data deleted in "+time.Since(timer).String())

}

func DeleteDownloadedFile(deviceFileName string, device database.Device) {
	deviceDirectory := filepath.Join(serviceDirectory, strconv.FormatUint(uint64(device.ID), 10)+"-"+device.Name)
	deviceFullPath := strings.Join([]string{deviceDirectory, deviceFileName}, "/")
	info, err := os.Stat(deviceFullPath)
	if err != nil {
		LogError(device.Name, "File does not exist: "+err.Error())
		return
	}
	if !info.IsDir() {
		err := os.Remove(deviceFullPath)
		if err != nil {
			LogError(device.Name, "Problem deleting file, "+err.Error())
		}
	}
}

func CheckActive(device database.Device) bool {
	for _, activeDevice := range activeDevices {
		if activeDevice.Name == device.Name {
			LogInfo(device.Name, "Device still active")
			return true
		}
	}
	LogInfo(device.Name, "Device not active")
	return false
}

func RemoveDeviceFromRunningDevices(device database.Device) {
	deviceSync.Lock()
	for idx, runningDevice := range runningDevices {
		if device.Name == runningDevice.Name {
			runningDevices = append(runningDevices[0:idx], runningDevices[idx+1:]...)
		}
	}
	deviceSync.Unlock()
}

func ReadActiveDevices(reference string) {
	LogInfo("MAIN", "Reading active devices")
	timer := time.Now()
	db, err := gorm.Open(postgres.Open(config), &gorm.Config{})
	if err != nil {
		LogError(reference, "Problem opening database: "+err.Error())
		activeDevices = nil
		return
	}
	sqlDB, err := db.DB()
	defer sqlDB.Close()
	var deviceType database.DeviceType
	db.Where("name=?", "Zapsi").Find(&deviceType)
	db.Where("device_type_id=?", deviceType.ID).Where("activated = true").Find(&activeDevices)
	LogInfo("MAIN", "Active devices read in "+time.Since(timer).String())
}
