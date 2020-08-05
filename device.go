package main

import (
	"bufio"
	"fmt"
	"github.com/PaesslerAG/gval"
	"github.com/dustin/go-humanize"
	"github.com/petrjahoda/database"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const downloadTimeoutInSeconds = 60

var deviceNameForDownload = ""
var processDownload uint64 = 0
var deviceFileDownloading = ""

const setZapsiTimeAtHour = 0
const setZapsiTimeAtMinute = 0

var tempPorts []tempPort

type BadDataError struct {
	data string
}

type tempPort struct {
	port  string
	value float32
}

func DownloadDataFromDevice(device database.Device) (downloaded bool, error error) {
	LogInfo(device.Name, "Downloading data")
	timer := time.Now()
	deviceNameForDownload = device.Name
	db, err := gorm.Open(postgres.Open(config), &gorm.Config{})
	if err != nil {
		LogError(device.Name, "Problem opening database: "+err.Error())
		return false, err
	}
	sqlDB, err := db.DB()
	defer sqlDB.Close()
	var digitalPorts []database.DevicePort
	var analogPorts []database.DevicePort
	var serialPorts []database.DevicePort
	var energyPorts []database.DevicePort
	db.Where("device_id = ?", device.ID).Where("device_port_type_id = ?", 1).Find(&digitalPorts)
	db.Where("device_id = ?", device.ID).Where("device_port_type_id = ?", 2).Find(&analogPorts)
	db.Where("device_id = ?", device.ID).Where("device_port_type_id = ?", 3).Find(&serialPorts)
	db.Where("device_id = ?", device.ID).Where("device_port_type_id = ?", 4).Find(&energyPorts)
	if len(digitalPorts) > 0 {
		LogInfo(device.Name, "Device has digital ports, downloading data...")
		fileUrl := "http://" + device.IpAddress + "/log/digital.txt"
		deviceDirectory := filepath.Join(serviceDirectory, strconv.FormatUint(uint64(device.ID), 10)+"-"+device.Name)
		deviceFileName := "digital.txt"
		deviceFullPath := strings.Join([]string{deviceDirectory, deviceFileName}, "/")
		deviceFileDownloading = fileUrl
		if err := DownloadFile(device.Name, deviceFullPath, fileUrl); err != nil {
			LogError(device.Name, fileUrl+" problem downloading "+err.Error())
		}

	}
	if len(analogPorts) > 0 {
		LogInfo(device.Name, "Device has analog ports, downloading data...")
		fileUrl := "http://" + device.IpAddress + "/log/analog.txt"
		deviceDirectory := filepath.Join(serviceDirectory, strconv.FormatUint(uint64(device.ID), 10)+"-"+device.Name)
		deviceFileName := "analog.txt"
		deviceFullPath := strings.Join([]string{deviceDirectory, deviceFileName}, "/")
		deviceFileDownloading = fileUrl
		if err := DownloadFile(device.Name, deviceFullPath, fileUrl); err != nil {
			LogError(device.Name, fileUrl+" problem downloading "+err.Error())
			KillPort(device)
		}

	}
	if len(serialPorts) > 0 {
		LogInfo(device.Name, "Device has serial ports, downloading data...")
		fileUrl := "http://" + device.IpAddress + "/log/serial.txt"
		deviceDirectory := filepath.Join(serviceDirectory, strconv.FormatUint(uint64(device.ID), 10)+"-"+device.Name)
		deviceFileName := "serial.txt"
		deviceFullPath := strings.Join([]string{deviceDirectory, deviceFileName}, "/")
		deviceFileDownloading = fileUrl
		if err := DownloadFile(device.Name, deviceFullPath, fileUrl); err != nil {
			LogError(device.Name, fileUrl+" problem downloading "+err.Error())
		}

	}
	if len(energyPorts) > 0 {
		LogInfo(device.Name, "Device has energy ports, downloading data...")
		fileUrl := "http://" + device.IpAddress + "/log/ui_value.txt"
		deviceDirectory := filepath.Join(serviceDirectory, strconv.FormatUint(uint64(device.ID), 10)+"-"+device.Name)
		deviceFileName := "ui_value.txt"
		deviceFullPath := strings.Join([]string{deviceDirectory, deviceFileName}, "/")
		deviceFileDownloading = fileUrl
		if err := DownloadFile(device.Name, deviceFullPath, fileUrl); err != nil {
			LogError(device.Name, fileUrl+" problem downloading "+err.Error())
		}

	}
	deviceFileDownloading = ""
	LogInfo(device.Name, "Data downloaded in "+time.Since(timer).String())
	return true, nil
}

func DownloadFile(deviceName string, filepath string, url string) error {
	LogInfo(deviceName, "Downloading file, process started: "+url)
	timer := time.Now()
	client := http.Client{
		Timeout: downloadTimeoutInSeconds * time.Second,
	}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	LogInfo(deviceName, url+" file size "+humanize.Bytes(uint64(int(resp.ContentLength))))
	defer resp.Body.Close()
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()
	counter := &WriteCounter{}
	_, err = io.Copy(out, io.TeeReader(resp.Body, counter))
	if err != nil {
		return err
	}
	processDownload = 0
	LogInfo(deviceName, url+" file downloaded "+humanize.Bytes(uint64(int(resp.ContentLength))))
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		LogError(deviceName, url+" cannot delete file : "+err.Error())
		return err
	} else {
		resp, err := client.Do(req)
		if err != nil {
			LogError(deviceName, url+" cannot delete file: "+err.Error())
			return err
		} else {
			LogInfo(deviceName, url+" file deleted")

		}
		defer resp.Body.Close()
	}
	LogInfo(deviceName, "Downloading file, process ended in "+time.Since(timer).String())
	return nil
}

type WriteCounter struct {
	Total uint64
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	wc.PrintProgress()
	return n, nil
}

func (wc WriteCounter) PrintProgress() {
	actualProcess := wc.Total / 500000
	if actualProcess != processDownload {
		LogInfo(deviceNameForDownload, deviceFileDownloading+" file downloaded: "+humanize.Bytes(wc.Total))
		processDownload = actualProcess
	}
}

func ProcessSortedData(device database.Device, intermediateData []SortedData) error {
	LogInfo(device.Name, "Processing data")
	timer := time.Now()
	db, err := gorm.Open(postgres.Open(config), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		LogError(device.Name, "Problem opening database: "+err.Error())
		return err
	}
	sqlDB, err := db.DB()
	defer sqlDB.Close()
	var digitalPorts []database.DevicePort
	db.Where("device_id = ?", device.ID).Where("device_port_type_id = ?", 1).Where("virtual = ?", false).Find(&digitalPorts)
	var analogPorts []database.DevicePort
	db.Where("device_id = ?", device.ID).Where("device_port_type_id = ?", 2).Where("virtual = ?", false).Find(&analogPorts)
	var serialPorts []database.DevicePort
	db.Where("device_id = ?", device.ID).Where("device_port_type_id = ?", 3).Where("virtual = ?", false).Find(&serialPorts)
	var energyPorts []database.DevicePort
	db.Where("device_id = ?", device.ID).Where("device_port_type_id = ?", 4).Where("virtual = ?", false).Find(&energyPorts)
	var virtualDigitalPorts []database.DevicePort
	db.Where("device_id = ?", device.ID).Where("device_port_type_id = ?", 1).Where("virtual = ?", true).Find(&virtualDigitalPorts)
	var virtualAnalogPorts []database.DevicePort
	db.Where("device_id = ?", device.ID).Where("device_port_type_id = ?", 2).Where("virtual = ?", true).Find(&virtualAnalogPorts)
	var virtualSerialPorts []database.DevicePort
	db.Where("device_id = ?", device.ID).Where("device_port_type_id = ?", 3).Where("virtual = ?", true).Find(&virtualSerialPorts)
	var virtualEnergyPorts []database.DevicePort
	db.Where("device_id = ?", device.ID).Where("device_port_type_id = ?", 4).Where("virtual = ?", true).Find(&virtualEnergyPorts)

	ReadTempPorts(analogPorts, db, digitalPorts, serialPorts, energyPorts, virtualAnalogPorts, virtualDigitalPorts, virtualSerialPorts, virtualEnergyPorts, device)

	var analogRecordsToInsert []database.DevicePortAnalogRecord
	var digitalRecordsToInsert []database.DevicePortDigitalRecord
	var serialRecordsToInsert []database.DevicePortSerialRecord
	for _, record := range intermediateData {
		switch record.Type {
		case digital:
			for _, port := range digitalPorts {
				digitalRecordsToInsert = append(digitalRecordsToInsert, SaveDigitalDataToDatabase(port, record, device))
			}
		case analog:
			for _, port := range analogPorts {
				analogRecordsToInsert = append(analogRecordsToInsert, SaveAnalogDataToDatabase(port, record, device))
			}
		case serial:
			for _, port := range serialPorts {
				serialRecordsToInsert = append(serialRecordsToInsert, SaveSerialDataToDatabase(port, record, device))
			}
		case energy:
			for _, port := range energyPorts {
				analogRecordsToInsert = append(analogRecordsToInsert, SaveEnergyDataToDatabase(port, record, device))
			}
		}

		if len(virtualDigitalPorts) > 0 {
			for _, port := range virtualDigitalPorts {
				digitalRecordsToInsert = SaveVirtualDigitalDataToDatabase(port, record, device, digitalRecordsToInsert)
			}
		}
		if len(virtualAnalogPorts) > 0 {
			for _, port := range virtualAnalogPorts {
				analogRecordsToInsert = append(analogRecordsToInsert, SaveVirtualAnalogDataToDatabase(port, record, device, db))
			}
		}
		if len(virtualSerialPorts) > 0 {
			for _, port := range virtualSerialPorts {
				serialRecordsToInsert = append(serialRecordsToInsert, SaveVirtualSerialDataToDatabase(port, record, device))
			}
		}
		if len(virtualEnergyPorts) > 0 {
			for _, port := range virtualEnergyPorts {
				analogRecordsToInsert = append(analogRecordsToInsert, SaveVirtualEnergyDataToDatabase(port, record, device))
			}
		}
		if len(analogRecordsToInsert) > 10000 {
			db.Clauses(clause.OnConflict{DoNothing: true}).Create(&analogRecordsToInsert)
			analogRecordsToInsert = nil
		}
		if len(digitalRecordsToInsert) > 10000 {
			db.Clauses(clause.OnConflict{DoNothing: true}).Create(&digitalRecordsToInsert)
			digitalRecordsToInsert = nil
		}
		if len(serialRecordsToInsert) > 10000 {
			db.Clauses(clause.OnConflict{DoNothing: true}).Create(&serialRecordsToInsert)
			serialRecordsToInsert = nil
		}

	}
	db.Clauses(clause.OnConflict{DoNothing: true}).Create(&analogRecordsToInsert)
	analogRecordsToInsert = nil
	db.Clauses(clause.OnConflict{DoNothing: true}).Create(&digitalRecordsToInsert)
	digitalRecordsToInsert = nil
	db.Clauses(clause.OnConflict{DoNothing: true}).Create(&serialRecordsToInsert)
	serialRecordsToInsert = nil
	LogInfo(device.Name, "Data processed in "+time.Since(timer).String())
	return nil
}

func SaveVirtualDigitalDataToDatabase(port database.DevicePort, record SortedData, device database.Device, digitalRecordsToInsert []database.DevicePortDigitalRecord) []database.DevicePortDigitalRecord {
	if strings.Contains(port.Settings, "SP:ADDZERO") {
		digitalRecordsToInsert = ProcessAddZeroPort(device, record, port, digitalRecordsToInsert)
	} else {
		recordToInsert := ProcessDataAsStandardVirtualDigitalPort(port, record, device)
		if !recordToInsert.DateTime.IsZero() {
			digitalRecordsToInsert = append(digitalRecordsToInsert, recordToInsert)
		}
	}
	return digitalRecordsToInsert
}

func ProcessAddZeroPort(device database.Device, record SortedData, port database.DevicePort, digitalRecordsToInsert []database.DevicePortDigitalRecord) []database.DevicePortDigitalRecord {
	var recordToInsert database.DevicePortDigitalRecord
	if record.Type == digital {
		originalPort := port.Settings[12 : len(port.Settings)-1]
		positionInFile, err := strconv.Atoi(originalPort)
		if err != nil {
			LogError(device.Name, "Problem parsing settings from port D"+originalPort+": "+err.Error())
		}
		parsedData := strings.Split(record.RawData, ";")
		dataToInsert, err := strconv.Atoi(parsedData[positionInFile-1])
		if err != nil {
			LogError(device.Name, "Problem parsing settings from port "+port.Name+" ["+port.Settings+"]: "+err.Error())
		}
		if dataToInsert == 1 {
			for index, tempPort := range tempPorts {
				portPrefix := "D"
				if port.PortNumber < 10 {
					portPrefix += "0"
				}
				if tempPort.port == portPrefix+strconv.Itoa(port.PortNumber) {
					if int(tempPort.value) != dataToInsert {
						dateTimeToInsert := record.DateTime
						recordToInsert = database.DevicePortDigitalRecord{DateTime: dateTimeToInsert, Data: dataToInsert, DevicePortID: int(port.ID)}
						digitalRecordsToInsert = append(digitalRecordsToInsert, recordToInsert)
						dataToInsert = 0
						dateTimeToInsert = record.DateTime.Add(100 * time.Millisecond)
						recordToInsert = database.DevicePortDigitalRecord{DateTime: dateTimeToInsert, Data: dataToInsert, DevicePortID: int(port.ID)}
						digitalRecordsToInsert = append(digitalRecordsToInsert, recordToInsert)
						tempPorts[index].value = float32(dataToInsert)
					}
				}
			}
		}
	}
	return digitalRecordsToInsert
}

func ReadTempPorts(analogPorts []database.DevicePort, db *gorm.DB, digitalPorts []database.DevicePort, serialPorts []database.DevicePort, energyPorts []database.DevicePort, virtualAnalogPorts []database.DevicePort, virtualDigitalPorts []database.DevicePort, virtualSerialPorts []database.DevicePort, virtualEnergyPorts []database.DevicePort, device database.Device) {
	LogInfo(device.Name, "Reading temp ports")
	timer := time.Now()
	for _, port := range analogPorts {
		var data database.DevicePortAnalogRecord
		db.Where("device_port_id = ?", port.ID).Last(&data)
		var tempPort tempPort
		if port.PortNumber < 10 {
			tempPort.port = "A0" + strconv.Itoa(port.PortNumber)
		} else {
			tempPort.port = "A" + strconv.Itoa(port.PortNumber)
		}
		tempPort.value = data.Data
		tempPorts = append(tempPorts, tempPort)
	}
	for _, port := range digitalPorts {
		var data database.DevicePortDigitalRecord
		db.Where("device_port_id = ?", port.ID).Last(&data)
		var tempPort tempPort
		if port.PortNumber < 10 {
			tempPort.port = "D0" + strconv.Itoa(port.PortNumber)
		} else {
			tempPort.port = "D" + strconv.Itoa(port.PortNumber)
		}
		tempPort.value = float32(data.Data)
		tempPorts = append(tempPorts, tempPort)
	}
	for _, port := range serialPorts {
		var data database.DevicePortSerialRecord
		db.Where("device_port_id = ?", port.ID).Last(&data)
		var tempPort tempPort
		if port.PortNumber < 10 {
			tempPort.port = "S0" + strconv.Itoa(port.PortNumber)
		} else {
			tempPort.port = "S" + strconv.Itoa(port.PortNumber)
		}
		tempPort.value = data.Data
		tempPorts = append(tempPorts, tempPort)
	}
	for _, port := range energyPorts {
		var data database.DevicePortAnalogRecord
		db.Where("device_port_id = ?", port.ID).Last(&data)
		var tempPort tempPort
		if port.PortNumber < 10 {
			tempPort.port = "E0" + strconv.Itoa(port.PortNumber)
		} else {
			tempPort.port = "E" + strconv.Itoa(port.PortNumber)
		}
		tempPort.value = data.Data
		tempPorts = append(tempPorts, tempPort)
	}

	for _, port := range virtualAnalogPorts {
		var data database.DevicePortAnalogRecord
		db.Where("device_port_id = ?", port.ID).Last(&data)
		var tempPort tempPort
		if port.PortNumber < 10 {
			tempPort.port = "A0" + strconv.Itoa(port.PortNumber)
		} else {
			tempPort.port = "A" + strconv.Itoa(port.PortNumber)
		}
		tempPort.value = data.Data
		tempPorts = append(tempPorts, tempPort)
	}
	for _, port := range virtualDigitalPorts {
		var data database.DevicePortDigitalRecord
		db.Where("device_port_id = ?", port.ID).Last(&data)
		var tempPort tempPort
		if port.PortNumber < 10 {
			tempPort.port = "D0" + strconv.Itoa(port.PortNumber)
		} else {
			tempPort.port = "D" + strconv.Itoa(port.PortNumber)
		}
		tempPort.value = float32(data.Data)
		tempPorts = append(tempPorts, tempPort)
	}
	for _, port := range virtualSerialPorts {
		var data database.DevicePortSerialRecord
		db.Where("device_port_id = ?", port.ID).Last(&data)
		var tempPort tempPort
		if port.PortNumber < 10 {
			tempPort.port = "S0" + strconv.Itoa(port.PortNumber)
		} else {
			tempPort.port = "S" + strconv.Itoa(port.PortNumber)
		}
		tempPort.value = data.Data
		tempPorts = append(tempPorts, tempPort)
	}
	for _, port := range virtualEnergyPorts {
		var data database.DevicePortAnalogRecord
		db.Where("device_port_id = ?", port.ID).Last(&data)
		var tempPort tempPort
		if port.PortNumber < 10 {
			tempPort.port = "E0" + strconv.Itoa(port.PortNumber)
		} else {
			tempPort.port = "E" + strconv.Itoa(port.PortNumber)
		}
		tempPort.value = data.Data
		tempPorts = append(tempPorts, tempPort)
	}
	LogInfo(device.Name, "Temp ports read in "+time.Since(timer).String())
}

func SaveVirtualEnergyDataToDatabase(port database.DevicePort, record SortedData, device database.Device) database.DevicePortAnalogRecord {
	var recordToInsert database.DevicePortAnalogRecord
	result := ReplacePortNameWithItsValue(port.Settings)
	value, err := gval.Evaluate(result, nil)
	if err != nil {
		LogError(device.Name, "Problem evaluating data: "+err.Error())
		return recordToInsert
	}
	dataToInsert := float32(value.(float64))
	for index, tempPort := range tempPorts {
		portPrefix := "E"
		if port.PortNumber < 10 {
			portPrefix += "0"
		}
		if tempPort.port == portPrefix+strconv.Itoa(port.PortNumber) {
			dateTimeToInsert := record.DateTime
			recordToInsert = database.DevicePortAnalogRecord{DateTime: dateTimeToInsert, Data: dataToInsert, DevicePortID: int(port.ID)}
			tempPorts[index].value = dataToInsert
			break
		}
	}
	return recordToInsert
}

func SaveVirtualSerialDataToDatabase(port database.DevicePort, record SortedData, device database.Device) database.DevicePortSerialRecord {
	var recordToInsert database.DevicePortSerialRecord
	result := ReplacePortNameWithItsValue(port.Settings)
	value, err := gval.Evaluate(result, nil)
	if err != nil {
		LogError(device.Name, "Problem evaluating data: "+err.Error())
		return recordToInsert
	}
	dataToInsert := float32(value.(float64))
	for index, tempPort := range tempPorts {
		portPrefix := "S"
		if port.PortNumber < 10 {
			portPrefix += "0"
		}
		if tempPort.port == portPrefix+strconv.Itoa(port.PortNumber) {
			dateTimeToInsert := record.DateTime
			recordToInsert = database.DevicePortSerialRecord{DateTime: dateTimeToInsert, Data: dataToInsert, DevicePortID: int(port.ID)}
			tempPorts[index].value = dataToInsert
			break
		}
	}
	return recordToInsert
}

func SaveVirtualAnalogDataToDatabase(port database.DevicePort, record SortedData, device database.Device, db *gorm.DB) database.DevicePortAnalogRecord {
	var recordToInsert database.DevicePortAnalogRecord
	if strings.Contains(port.Settings, "SP:TC") {
		recordToInsert = ProcessThermoCouplePort(record, port, db, device)
	} else if strings.Contains(port.Settings, "SP:SPEED") {
		recordToInsert = ProcessSpeedPort(record, port, db, device)
	} else {
		recordToInsert = ProcessDataAsStandardVirtualAnalogPort(record, port, device)
	}
	return recordToInsert
}

func ProcessThermoCouplePort(record SortedData, port database.DevicePort, db *gorm.DB, device database.Device) database.DevicePortAnalogRecord {
	var recordToInsert database.DevicePortAnalogRecord
	parameters := strings.Split(port.Settings[6:len(port.Settings)-1], ";")
	thermoCoupleType := parameters[0]
	thermoCoupleMainPortId := parameters[1][1:]
	thermoCoupleColdJunctionPortId := parameters[2][1:]
	thermoCoupleTypeId := SelectThermoCouple(thermoCoupleType)
	recordToInsert = ProcessThermoCouplePortData(record, thermoCoupleMainPortId, thermoCoupleColdJunctionPortId, thermoCoupleTypeId, port, db, device)
	return recordToInsert
}

func ProcessThermoCouplePortData(record SortedData, thermoCoupleMainPortId string, thermoCoupleColdJunctionPortId string, thermoCoupleTypeId int, port database.DevicePort, db *gorm.DB, device database.Device) database.DevicePortAnalogRecord {
	var recordToInsert database.DevicePortAnalogRecord
	var thermoCoupleMainPort database.DevicePort
	var thermoCoupleColdJunctionPort database.DevicePort
	db.Where("device_id = ?", device.ID).Where("port_number = ?", thermoCoupleMainPortId).Find(&thermoCoupleMainPort)
	db.Where("device_id = ?", device.ID).Where("port_number = ?", thermoCoupleColdJunctionPortId).Find(&thermoCoupleColdJunctionPort)

	var thermoCoupleMainPortData float32
	for _, tempPort := range tempPorts {
		portPrefix := "A"
		if port.PortNumber < 10 {
			portPrefix += "0"
		}
		if tempPort.port == portPrefix+strconv.Itoa(thermoCoupleMainPort.PortNumber) {
			thermoCoupleMainPortData = tempPort.value
			break
		}
	}
	dataAsMv := math.Abs(float64(thermoCoupleMainPortData)) / 8.0 * 0.015625
	value := float32(ConvertMvToTemp(dataAsMv, thermoCoupleTypeId))
	var coldJunctionTemperature float32
	for _, tempPort := range tempPorts {
		if tempPort.port == "A"+strconv.Itoa(thermoCoupleColdJunctionPort.PortNumber) {
			coldJunctionTemperature = tempPort.value
			break
		}
	}
	value = value + coldJunctionTemperature
	dataToInsert := value
	for index, tempPort := range tempPorts {
		if tempPort.port == "A"+strconv.Itoa(port.PortNumber) {
			dateTimeToInsert := record.DateTime
			recordToInsert = database.DevicePortAnalogRecord{DateTime: dateTimeToInsert, Data: dataToInsert, DevicePortID: int(port.ID)}
			tempPorts[index].value = dataToInsert
			break
		}
	}
	return recordToInsert
}

func ProcessSpeedPort(record SortedData, port database.DevicePort, db *gorm.DB, device database.Device) database.DevicePortAnalogRecord {
	var recordToInsert database.DevicePortAnalogRecord
	speed, err := CalculateSpeed(device, port, db)
	if err != nil {
		LogError(device.Name, "Problem evaluating data for speed port: "+err.Error())
		return recordToInsert
	}
	dataToInsert := speed
	for index, tempPort := range tempPorts {
		portPrefix := "A"
		if port.PortNumber < 10 {
			portPrefix += "0"
		}
		if tempPort.port == portPrefix+strconv.Itoa(port.PortNumber) {
			dateTimeToInsert := record.DateTime
			recordToInsert = database.DevicePortAnalogRecord{DateTime: dateTimeToInsert, Data: dataToInsert, DevicePortID: int(port.ID)}
			tempPorts[index].value = dataToInsert
			break
		}
	}
	return recordToInsert
}

func CalculateSpeed(device database.Device, virtualPort database.DevicePort, db *gorm.DB) (float32, error) {
	parameters := strings.Split(virtualPort.Settings[9:len(virtualPort.Settings)-1], ";")
	port := parameters[0]
	minutesBack := parameters[1]
	diameterAsString := parameters[2]
	portNumber := port[1:]
	minutes, err := strconv.Atoi(minutesBack)
	if err != nil {
		return 0, err
	}
	diameter, err := strconv.ParseFloat(diameterAsString, 64)
	if err != nil {
		return 0, err
	}
	timeForData := time.Now().UTC().Add(time.Duration(minutes) * time.Minute)
	var devicePort database.DevicePort
	db.Where("device_id = ?", device.ID).Where("port_number = ?", portNumber).Find(&devicePort)
	var digitalRecords []database.DevicePortDigitalRecord
	db.Where("device_port_id = ?", devicePort.ID).Where("date_time > ?", timeForData).Where("data = ?", 0).Find(&digitalRecords)
	speed := float32(len(digitalRecords)) * float32(diameter) * math.Pi
	return speed, nil
}

func ProcessDataAsStandardVirtualAnalogPort(record SortedData, port database.DevicePort, device database.Device) database.DevicePortAnalogRecord {
	var recordToInsert database.DevicePortAnalogRecord
	result := ReplacePortNameWithItsValue(port.Settings)
	value, err := gval.Evaluate(result, nil)
	if err != nil {
		LogError(device.Name, "Problem evaluating data: "+err.Error())
		return recordToInsert
	}
	dataToInsert := float32(value.(float64))
	for index, tempPort := range tempPorts {
		portPrefix := "A"
		if port.PortNumber < 10 {
			portPrefix += "0"
		}
		if tempPort.port == portPrefix+strconv.Itoa(port.PortNumber) {
			dateTimeToInsert := record.DateTime
			recordToInsert = database.DevicePortAnalogRecord{DateTime: dateTimeToInsert, Data: dataToInsert, DevicePortID: int(port.ID)}
			tempPorts[index].value = dataToInsert
			break
		}
	}
	return recordToInsert
}

func ProcessDataAsStandardVirtualDigitalPort(port database.DevicePort, record SortedData, device database.Device) database.DevicePortDigitalRecord {
	var recordToInsert database.DevicePortDigitalRecord
	result := ReplacePortNameWithItsValue(port.Settings)
	value, err := gval.Evaluate(result, nil)
	if err != nil {
		LogError(device.Name, "Problem evaluating data: "+err.Error())
		return recordToInsert
	}
	for index, tempPort := range tempPorts {
		portPrefix := "D"
		if port.PortNumber < 10 {
			portPrefix += "0"
		}
		if tempPort.port == portPrefix+strconv.Itoa(port.PortNumber) {
			dataToInsert := 0
			if value.(bool) == true {
				dataToInsert = 1
			}
			if int(tempPort.value) != dataToInsert {
				dateTimeToInsert := record.DateTime
				recordToInsert = database.DevicePortDigitalRecord{DateTime: dateTimeToInsert, Data: dataToInsert, DevicePortID: int(port.ID)}
				tempPorts[index].value = float32(dataToInsert)
				break
			} else {
				LogWarning(device.Name, "Digital data mismatch, trying to save similar data to database: "+strconv.Itoa(dataToInsert))
				break
			}
		}
	}
	return recordToInsert
}

func ReplacePortNameWithItsValue(settings string) string {
	for _, port := range tempPorts {
		replacedValue := strconv.FormatFloat(float64(port.value), 'g', 15, 64)
		if strings.Contains(port.port, "D") {
			settings = strings.ReplaceAll(settings, port.port, replacedValue)
		} else if strings.Contains(port.port, "A") {
			settings = strings.ReplaceAll(settings, port.port, replacedValue)
		} else if strings.Contains(port.port, "S") {
			settings = strings.ReplaceAll(settings, port.port, replacedValue)
		} else if strings.Contains(port.port, "E") {
			settings = strings.ReplaceAll(settings, port.port, replacedValue)
		}
	}
	return settings
}

func PrepareDownloadedData(device database.Device) []SortedData {
	LogInfo(device.Name, "Preparing downloaded data")
	timer := time.Now()
	var sortedData []SortedData
	if FileExists("digital.txt", device) {
		AddDataForProcessing("digital.txt", &sortedData, device)
	}
	if FileExists("analog.txt", device) {
		AddDataForProcessing("analog.txt", &sortedData, device)
	}
	if FileExists("serial.txt", device) {
		AddDataForProcessing("serial.txt", &sortedData, device)
	}
	if FileExists("ui_value.txt", device) {
		AddDataForProcessing("ui_value.txt", &sortedData, device)
	}
	sort.Slice(sortedData, func(i, j int) bool {
		return sortedData[i].DateTime.Before(sortedData[j].DateTime)
	})
	LogInfo(device.Name, "Data sorted, number of records: "+strconv.Itoa(len(sortedData)))
	LogInfo(device.Name, "Data prepared in "+time.Since(timer).String())
	return sortedData
}

func SaveEnergyDataToDatabase(port database.DevicePort, record SortedData, device database.Device) database.DevicePortAnalogRecord {
	var recordToInsert database.DevicePortAnalogRecord
	positionInFile := port.PortNumber - 1
	parsedData := strings.Split(record.RawData, ";")
	dataToInsert, err := strconv.ParseFloat(parsedData[positionInFile], 32)
	if err != nil {
		LogError(device.Name, "Problem parsing record: "+err.Error())
		return recordToInsert
	}
	for index, tempPort := range tempPorts {
		portPrefix := "E"
		if port.PortNumber < 10 {
			portPrefix += "0"
		}
		if tempPort.port == portPrefix+strconv.Itoa(port.PortNumber) {
			dateTimeToInsert := record.DateTime
			recordToInsert = database.DevicePortAnalogRecord{DateTime: dateTimeToInsert, Data: float32(dataToInsert), DevicePortID: int(port.ID)}
			tempPorts[index].value = float32(dataToInsert)
			break
		}
	}
	return recordToInsert
}

func SaveSerialDataToDatabase(port database.DevicePort, record SortedData, device database.Device) database.DevicePortSerialRecord {
	var recordToInsert database.DevicePortSerialRecord
	positionInFile := port.PortNumber - 1
	parsedData := strings.Split(record.RawData, ";")
	dataToInsert, err := strconv.ParseFloat(parsedData[positionInFile], 32)
	if err != nil {
		LogError(device.Name, "Problem parsing record: "+err.Error())
		return recordToInsert
	}
	for index, tempPort := range tempPorts {
		portPrefix := "S"
		if port.PortNumber < 10 {
			portPrefix += "0"
		}
		if tempPort.port == portPrefix+strconv.Itoa(port.PortNumber) {
			dateTimeToInsert := record.DateTime
			recordToInsert = database.DevicePortSerialRecord{DateTime: dateTimeToInsert, Data: float32(dataToInsert), DevicePortID: int(port.ID)}
			tempPorts[index].value = float32(dataToInsert)
			break
		}
	}
	return recordToInsert
}

func SaveDigitalDataToDatabase(port database.DevicePort, record SortedData, device database.Device) database.DevicePortDigitalRecord {
	var recordToInsert database.DevicePortDigitalRecord
	positionInFile := port.PortNumber - 1
	parsedData := strings.Split(record.RawData, ";")
	dataToInsert, err := strconv.Atoi(parsedData[positionInFile])
	if err != nil {
		LogError(device.Name, "Problem parsing record: "+err.Error())
		return recordToInsert
	}
	for index, tempPort := range tempPorts {
		portPrefix := "D"
		if port.PortNumber < 10 {
			portPrefix += "0"
		}
		if tempPort.port == portPrefix+strconv.Itoa(port.PortNumber) {
			if int(tempPort.value) != dataToInsert {
				dateTimeToInsert := record.DateTime
				recordToInsert = database.DevicePortDigitalRecord{DateTime: dateTimeToInsert, Data: dataToInsert, DevicePortID: int(port.ID)}
				tempPorts[index].value = float32(dataToInsert)
				break
			} else {
				LogWarning(device.Name, "Digital data mismatch, trying to save similar data to database: "+strconv.Itoa(dataToInsert))
				break
			}
		}
	}
	return recordToInsert
}

func SaveAnalogDataToDatabase(port database.DevicePort, record SortedData, device database.Device) database.DevicePortAnalogRecord {
	var recordToInsert database.DevicePortAnalogRecord
	positionInFile := port.PortNumber - 1
	parsedData := strings.Split(record.RawData, ";")
	dataToInsert, err := strconv.ParseFloat(parsedData[positionInFile], 32)
	if err != nil {
		LogError(device.Name, "Problem parsing record: "+err.Error())
		return recordToInsert
	}
	for index, tempPort := range tempPorts {
		portPrefix := "A"
		if port.PortNumber < 10 {
			portPrefix += "0"
		}
		if tempPort.port == portPrefix+strconv.Itoa(port.PortNumber) {
			dateTimeToInsert := record.DateTime
			recordToInsert = database.DevicePortAnalogRecord{DateTime: dateTimeToInsert, Data: float32(dataToInsert), DevicePortID: int(port.ID)}
			tempPorts[index].value = float32(dataToInsert)
			break
		}
	}

	return recordToInsert
}

func FileExists(filename string, device database.Device) bool {
	deviceDirectory := filepath.Join(serviceDirectory, strconv.FormatUint(uint64(device.ID), 10)+"-"+device.Name)
	deviceFullPath := strings.Join([]string{deviceDirectory, filename}, "/")
	if _, err := os.Stat(deviceFullPath); err == nil {
		return true
	} else if os.IsNotExist(err) {
		return false
	} else {
		return false
	}
}

func AddDataForProcessing(filename string, data *[]SortedData, device database.Device) {
	LogInfo(device.Name, "Adding data for processing: "+filename)
	timer := time.Now()
	deviceDirectory := filepath.Join(serviceDirectory, strconv.FormatUint(uint64(device.ID), 10)+"-"+device.Name)
	deviceFullPath := strings.Join([]string{deviceDirectory, filename}, "/")
	f, _ := os.Open(deviceFullPath)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		zapsiData := scanner.Text()
		parsedData := strings.Split(zapsiData, ";")
		rawData := parsedData[0]
		for i := 1; i < len(parsedData)-2; i++ {
			rawData += ";" + parsedData[i]
		}
		finalDateTime, err := ParseDateTimeFromData(parsedData)
		if err != nil {
			LogError(device.Name, "Problem parsing datetime from ["+zapsiData+"]: "+err.Error())
			continue
		}
		AddIntermediateData(finalDateTime, rawData, filename, data)
	}
	LogInfo(device.Name, "Data added in "+time.Since(timer).String())

}

func AddIntermediateData(finalDateTime time.Time, rawData string, filename string, data *[]SortedData) {
	dataForInsert := SortedData{DateTime: finalDateTime, RawData: rawData}
	switch filename {
	case "analog.txt":
		dataForInsert.Type = analog
	case "digital.txt":
		dataForInsert.Type = digital
	case "serial.txt":
		dataForInsert.Type = serial
	case "ui_value.txt":
		dataForInsert.Type = energy
	}
	*data = append(*data, dataForInsert)
}

func ParseDateTimeFromData(data []string) (time.Time, error) {
	if len(data) > 1 {
		dataDate := strings.Split(data[len(data)-2], ".")
		dataDay := dataDate[0]
		dataMonth := dataDate[1]
		dataYear := dataDate[2]
		intermediateDataTime := data[len(data)-1]
		var dataHour string
		var dataMinute string
		var dataSecond string
		var dataMilliSecond string
		if strings.Contains(intermediateDataTime, ".") {
			dataTime := strings.Split(intermediateDataTime, ".")
			dataTimeWithoutMillisecond := strings.Split(dataTime[0], ":")
			dataHour = dataTimeWithoutMillisecond[0]
			dataMinute = dataTimeWithoutMillisecond[1]
			dataSecond = dataTimeWithoutMillisecond[2]
			dataMilliSecond = dataTime[1]
		} else {
			dataTime := strings.Split(intermediateDataTime, ":")
			dataHour = dataTime[0]
			dataMinute = dataTime[1]
			dataSecond = dataTime[2]
			if len(dataTime) > 3 {
				dataMilliSecond = dataTime[3]
			} else {
				dataMilliSecond = "0"
			}
		}
		switch len(dataMilliSecond) {
		case 1:
			dataMilliSecond = "00" + dataMilliSecond
		case 2:
			dataMilliSecond = "0" + dataMilliSecond
		}
		input := dataYear + "-" + dataMonth + "-" + dataDay + " " + dataHour + ":" + dataMinute + ":" + dataSecond + "." + dataMilliSecond
		layout := "2006-1-2 15:4:5.000"

		finalDateTime, err := time.Parse(layout, input)
		return finalDateTime, err
	}
	return time.Now(), BadDataError{}
}

func (e BadDataError) Error() string {
	return fmt.Sprintf("bad line in input data")
}

func SendUDP(device database.Device, dstIP string, dstPort int, localIP string, localPort uint, data []byte) {
	RemoteEP := net.UDPAddr{IP: net.ParseIP(dstIP), Port: dstPort}
	localAddrString := fmt.Sprintf("%s:%d", localIP, localPort)
	LocalAddr, err := net.ResolveUDPAddr("udp", localAddrString)
	if err != nil {
		LogError(device.Name, "UDP problem: "+err.Error())
		return
	}

	conn, err := net.DialUDP("udp", LocalAddr, &RemoteEP)
	if err != nil {
		LogError(device.Name, "UDP creating problem: "+err.Error())
		return
	}
	LogInfo(device.Name, "UDP connection opened")
	result, err := conn.Write(data)
	if err != nil {
		LogError(device.Name, "UDP writing error: "+err.Error())
		return
	}
	LogInfo(device.Name, "UDP data written to Zapsi: "+string(data)+", with result of "+strconv.Itoa(result))
	closingUdpError := conn.Close()
	if closingUdpError != nil {
		LogError(device.Name, "UDP closing problem: "+closingUdpError.Error())
		return
	}
	LogInfo(device.Name, "UDP connection closed")
}
func SendTimeToDeviceAtStart(device database.Device) (timeUpdated bool) {
	LogInfo(device.Name, "Sending time to device")
	timer := time.Now()
	dateTimeForZapsi := time.Now().UTC().Format("02.01.2006 15:04:05")
	dateTimeForZapsi = "set_datetime=" + dateTimeForZapsi + " 0" + strconv.Itoa(int(time.Now().UTC().Weekday())) + "&"
	SendUDP(device, device.IpAddress, 9999, "", 0, []byte(dateTimeForZapsi))
	LogInfo(device.Name, "Time to device sent in "+time.Since(timer).String())
	return true
}

func KillPort(device database.Device) (timeUpdated bool) {
	LogInfo(device.Name, "Killing port 80")
	timer := time.Now()
	dateTimeForZapsi := time.Now().UTC().Format("02.01.2006 15:04:05")
	dateTimeForZapsi = "Kill80"
	SendUDP(device, device.IpAddress, 9999, "", 0, []byte(dateTimeForZapsi))
	LogInfo(device.Name, "Port 80 killed in "+time.Since(timer).String())
	return true
}

func SendTimeToDevice(device database.Device, timeUpdated bool) bool {
	LogInfo(device.Name, "Sending time to device")
	timer := time.Now()
	now := time.Now().UTC()
	dateTimeForZapsi := now.Format("02.01.2006 15:04:05")

	if now.Hour() == setZapsiTimeAtHour && now.Minute() == setZapsiTimeAtMinute && !timeUpdated {
		dateTimeForZapsi = "set_datetime=" + dateTimeForZapsi + " 0" + strconv.Itoa(int(now.Weekday())) + "&"
		SendUDP(device, device.IpAddress, 9999, "", 0, []byte(dateTimeForZapsi))
		return true
	}
	LogInfo(device.Name, "Time to device sent in "+time.Since(timer).String())
	if now.Hour() == setZapsiTimeAtHour && now.Minute() == setZapsiTimeAtMinute && timeUpdated {
		return true
	}
	return false
}
