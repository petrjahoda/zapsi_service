package main

import (
	"github.com/jinzhu/gorm"
	"github.com/petrjahoda/zapsi_database"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const version = "2020.1.2.2"
const programName = "Zapsi Service"
const deleteLogsAfter = 240 * time.Hour
const downloadInSeconds = 10

var (
	activeDevices  []zapsi_database.Device
	runningDevices []zapsi_database.Device
	deviceSync     sync.Mutex
)

func main() {
	LogDirectoryFileCheck("MAIN")
	LogInfo("MAIN", programName+" version "+version+" started")
	CreateConfigIfNotExists()
	LoadSettingsFromConfigFile()
	LogDebug("MAIN", "Using ["+DatabaseType+"] on "+DatabaseIpAddress+":"+DatabasePort+" with database "+DatabaseName)
	CompleteDatabaseCheck()
	for {
		start := time.Now()
		LogInfo("MAIN", "Program running")
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

func CompleteDatabaseCheck() {
	firstRunCheckComplete := false
	for firstRunCheckComplete == false {
		databaseOk := zapsi_database.CheckDatabase(DatabaseType, DatabaseIpAddress, DatabasePort, DatabaseLogin, DatabaseName, DatabasePassword)
		tablesOk, err := zapsi_database.CheckTables(DatabaseType, DatabaseIpAddress, DatabasePort, DatabaseLogin, DatabaseName, DatabasePassword)
		if err != nil {
			LogInfo("MAIN", "Problem creating tables: "+err.Error())
		}
		if databaseOk && tablesOk {
			WriteProgramVersionIntoSettings()
			firstRunCheckComplete = true
		}
	}
}

func WriteProgramVersionIntoSettings() {
	connectionString, dialect := zapsi_database.CheckDatabaseType(DatabaseType, DatabaseIpAddress, DatabasePort, DatabaseLogin, DatabaseName, DatabasePassword)
	db, err := gorm.Open(dialect, connectionString)
	if err != nil {
		LogError("MAIN", "Problem opening "+DatabaseName+" database: "+err.Error())
		activeDevices = nil
		return
	}
	defer db.Close()
	var settings zapsi_database.Setting
	db.Where("name=?", programName).Find(&settings)
	settings.Name = programName
	settings.Value = version
	db.Save(&settings)
	LogDebug("MAIN", "Updated version in database for "+programName)
}

func CheckDevice(device zapsi_database.Device) bool {
	for _, runningDevice := range runningDevices {
		if runningDevice.Name == device.Name {
			return true
		}
	}
	return false
}

func RunDevice(device zapsi_database.Device) {
	LogInfo(device.Name, "Device started running")
	deviceSync.Lock()
	runningDevices = append(runningDevices, device)
	deviceSync.Unlock()
	deviceIsActive := true
	CreateDirectoryIfNotExists(device)
	SendTime(device)
	timeUpdatedInLoop := false
	for deviceIsActive {
		start := time.Now()
		ProcessDownloadedFiles(device)
		success, err := DownloadData(device)
		if err != nil {
			LogError(device.Name, "Error downloading data: "+err.Error())
		}
		if success {
			ProcessDownloadedFiles(device)
		}
		LogInfo(device.Name, "Processing takes "+time.Since(start).String())
		timeUpdatedInLoop = SendTimeToZapsi(device, timeUpdatedInLoop)
		Sleep(device, start)

		deviceIsActive = CheckActive(device)
	}
	RemoveDeviceFromRunningDevices(device)
	LogInfo(device.Name, "Device not active, stopped running")

}

func CreateDirectoryIfNotExists(device zapsi_database.Device) {
	deviceDirectory := filepath.Join(".", strconv.Itoa(int(device.ID))+"-"+device.Name)

	if _, checkPathError := os.Stat(deviceDirectory); checkPathError == nil {
		LogInfo(device.Name, "Device directory exists")
	} else if os.IsNotExist(checkPathError) {
		LogWarning(device.Name, "Device directory not exist, creating")
		mkdirError := os.MkdirAll(deviceDirectory, 0777)
		if mkdirError != nil {
			LogError(device.Name, "Unable to create device directory: "+mkdirError.Error())
		} else {
			LogInfo(device.Name, "Device directory created")
		}
	} else {
		LogError(device.Name, "Device directory does not exist")
	}
}

func Sleep(device zapsi_database.Device, start time.Time) {
	if time.Since(start) < (downloadInSeconds * time.Second) {
		sleepTime := downloadInSeconds*time.Second - time.Since(start)
		LogInfo(device.Name, "Sleeping for "+sleepTime.String())
		time.Sleep(sleepTime)
	}
}

func ProcessDownloadedFiles(device zapsi_database.Device) {
	intermediateData := PrepareData(device)
	if len(intermediateData) > 0 {
		err := ProcessData(device, intermediateData)
		if err != nil {
			LogError(device.Name, "Error processing data: "+err.Error())
		}
	}
	DeleteDownloadedFile("digital.txt", device)
	DeleteDownloadedFile("analog.txt", device)
	DeleteDownloadedFile("serial.txt", device)
	DeleteDownloadedFile("ui_value.txt", device)
}

func DeleteDownloadedFile(deviceFileName string, device zapsi_database.Device) {
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

func CheckActive(device zapsi_database.Device) bool {
	for _, activeDevice := range activeDevices {
		if activeDevice.Name == device.Name {
			LogInfo(device.Name, "Device still active")
			return true
		}
	}
	LogInfo(device.Name, "Device not active")
	return false
}

func RemoveDeviceFromRunningDevices(device zapsi_database.Device) {
	deviceSync.Lock()
	for idx, runningDevice := range runningDevices {
		if device.Name == runningDevice.Name {
			runningDevices = append(runningDevices[0:idx], runningDevices[idx+1:]...)
		}
	}
	deviceSync.Unlock()
}

func UpdateActiveDevices(reference string) {
	connectionString, dialect := zapsi_database.CheckDatabaseType(DatabaseType, DatabaseIpAddress, DatabasePort, DatabaseLogin, DatabaseName, DatabasePassword)
	db, err := gorm.Open(dialect, connectionString)
	if err != nil {
		LogError(reference, "Problem opening "+DatabaseName+" database: "+err.Error())
		activeDevices = nil
		return
	}
	defer db.Close()
	var deviceType zapsi_database.DeviceType
	db.Where("name=?", "Zapsi").Find(&deviceType)
	db.Where("device_type_id=?", deviceType.ID).Where("activated = true").Find(&activeDevices)
	LogDebug("MAIN", "Zapsi device type id is "+strconv.Itoa(int(deviceType.ID)))
}
