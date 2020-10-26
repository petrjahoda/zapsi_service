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

func updateProgramVersion() {
	logInfo("MAIN", "Writing program version into settings")
	timer := time.Now()
	db, err := gorm.Open(postgres.Open(config), &gorm.Config{})
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	if err != nil {
		logError("MAIN", "Problem opening database: "+err.Error())
		return
	}
	var existingSettings database.Setting
	db.Where("name=?", serviceName).Find(&existingSettings)
	existingSettings.Name = serviceName
	existingSettings.Value = version
	db.Save(&existingSettings)
	logInfo("MAIN", "Program version written into settings in "+time.Since(timer).String())
}

func checkDeviceInRunningDevices(device database.Device) bool {
	for _, runningDevice := range runningDevices {
		if runningDevice.Name == device.Name {
			return true
		}
	}
	return false
}

func runDevice(device database.Device) {
	logInfo(device.Name, "Device active, started running")
	deviceSync.Lock()
	runningDevices = append(runningDevices, device)
	deviceSync.Unlock()
	deviceIsActive := true
	checkDeviceDataDirectory(device)
	sendTimeToDeviceAtStart(device)
	timeUpdatedInLoop := false
	for deviceIsActive && serviceRunning {
		logInfo(device.Name, "Device main loop started")
		timer := time.Now()
		dataSuccessfullyProcessed := processDownloadedData(device)
		if dataSuccessfullyProcessed {
			dataSuccessfullyDownloaded, err := downloadDataFromDevice(device)
			if err != nil {
				logError(device.Name, "Error downloading data: "+err.Error())
			}
			if dataSuccessfullyDownloaded {
				processDownloadedData(device)
			}
		}
		timeUpdatedInLoop = sendTimeToDevice(device, timeUpdatedInLoop)
		logInfo(device.Name, "Device main loop ended in "+time.Since(timer).String())
		sleep(device, timer)
		deviceIsActive = checkActive(device)
	}
	removeDeviceFromRunningDevices(device)
	logInfo(device.Name, "Device not active, stopped running")

}

func checkDeviceDataDirectory(device database.Device) {
	logInfo(device.Name, "Checking device data directory")
	timer := time.Now()
	deviceDirectory := filepath.Join(serviceDirectory, strconv.FormatUint(uint64(device.ID), 10)+"-"+device.Name)
	if _, checkPathError := os.Stat(deviceDirectory); checkPathError == nil {
		logInfo(device.Name, "Device directory already exists")
	} else if os.IsNotExist(checkPathError) {
		logError(device.Name, "Device directory not exist, creating")
		mkdirError := os.MkdirAll(deviceDirectory, 0777)
		if mkdirError != nil {
			logError(device.Name, "Unable to create device directory: "+mkdirError.Error())
		} else {
			logInfo(device.Name, "Device directory created")
		}
	} else {
		logError(device.Name, "Device directory does not exist")
	}
	logInfo(device.Name, "Device data directory checked in "+time.Since(timer).String())
}

func sleep(device database.Device, start time.Time) {
	if time.Since(start) < (downloadInSeconds * time.Second) {
		sleepTime := downloadInSeconds*time.Second - time.Since(start)
		logInfo(device.Name, "Sleeping for "+sleepTime.String())
		time.Sleep(sleepTime)
	}
}

func processDownloadedData(device database.Device) bool {
	logInfo(device.Name, "Processing downloaded data")
	timer := time.Now()
	sortedData := prepareDownloadedData(device)
	if len(sortedData) > 0 {
		err := processSortedData(device, sortedData)
		if err != nil {
			logError(device.Name, "Error processing data: "+err.Error())
			return false
		}
	}
	deleteDownloadedData(device)
	logInfo(device.Name, "Data processed in "+time.Since(timer).String())
	return true
}

func deleteDownloadedData(device database.Device) {
	logInfo(device.Name, "Deleting downloaded data")
	timer := time.Now()
	deleteDownloadedFile("digital.txt", device)
	deleteDownloadedFile("analog.txt", device)
	deleteDownloadedFile("serial.txt", device)
	deleteDownloadedFile("ui_value.txt", device)
	logInfo(device.Name, "Data deleted in "+time.Since(timer).String())

}

func deleteDownloadedFile(deviceFileName string, device database.Device) {
	deviceDirectory := filepath.Join(serviceDirectory, strconv.FormatUint(uint64(device.ID), 10)+"-"+device.Name)
	deviceFullPath := strings.Join([]string{deviceDirectory, deviceFileName}, "/")
	info, err := os.Stat(deviceFullPath)
	if err != nil {
		logError(device.Name, "File does not exist: "+err.Error())
		return
	}
	if !info.IsDir() {
		err := os.Remove(deviceFullPath)
		if err != nil {
			logError(device.Name, "Problem deleting file, "+err.Error())
		}
	}
}

func checkActive(device database.Device) bool {
	for _, activeDevice := range activeDevices {
		if activeDevice.Name == device.Name {
			logInfo(device.Name, "Device still active")
			return true
		}
	}
	logInfo(device.Name, "Device not active")
	return false
}

func removeDeviceFromRunningDevices(device database.Device) {
	deviceSync.Lock()
	for idx, runningDevice := range runningDevices {
		if device.Name == runningDevice.Name {
			runningDevices = append(runningDevices[0:idx], runningDevices[idx+1:]...)
		}
	}
	deviceSync.Unlock()
}

func readActiveDevices(reference string) {
	logInfo("MAIN", "Reading active devices")
	timer := time.Now()
	db, err := gorm.Open(postgres.Open(config), &gorm.Config{})
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	if err != nil {
		logError(reference, "Problem opening database: "+err.Error())
		activeDevices = nil
		return
	}
	var deviceType database.DeviceType
	db.Where("name=?", "Zapsi").Find(&deviceType)
	db.Where("device_type_id=?", deviceType.ID).Where("activated = true").Find(&activeDevices)
	logInfo("MAIN", "Active devices read in "+time.Since(timer).String())
}
