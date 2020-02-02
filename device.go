package main

import (
	"bufio"
	"fmt"
	"github.com/PaesslerAG/gval"
	"github.com/dustin/go-humanize"
	"github.com/jinzhu/gorm"
	"github.com/petrjahoda/zapsi_database"
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

type BadDataError struct {
	data string
}

func DownloadData(device zapsi_database.Device) (downloaded bool, error error) {
	deviceNameForDownload = device.Name
	connectionString, dialect := zapsi_database.CheckDatabaseType(DatabaseType, DatabaseIpAddress, DatabasePort, DatabaseLogin, DatabaseName, DatabasePassword)
	db, err := gorm.Open(dialect, connectionString)
	if err != nil {
		LogError(device.Name, "Problem opening "+DatabaseName+" database: "+err.Error())
		return false, err
	}
	var digitalPorts []zapsi_database.DevicePort
	var analogPorts []zapsi_database.DevicePort
	var serialPorts []zapsi_database.DevicePort
	var energyPorts []zapsi_database.DevicePort
	db.Where("device_id = ?", device.ID).Where("device_port_type_id = ?", 1).Find(&digitalPorts)
	db.Where("device_id = ?", device.ID).Where("device_port_type_id = ?", 2).Find(&analogPorts)
	db.Where("device_id = ?", device.ID).Where("device_port_type_id = ?", 3).Find(&serialPorts)
	db.Where("device_id = ?", device.ID).Where("device_port_type_id = ?", 4).Find(&energyPorts)
	LogDebug(device.Name, "Analog ports: "+strconv.Itoa(len(analogPorts))+", digital ports: "+strconv.Itoa(len(digitalPorts)))
	if len(digitalPorts) > 0 {
		LogInfo(device.Name, "Device has digital ports")
		fileUrl := "http://" + device.IpAddress + "/log/digital.txt"
		deviceDirectory := filepath.Join(".", strconv.Itoa(int(device.ID))+"-"+device.Name)
		deviceFileName := "digital.txt"
		deviceFullPath := strings.Join([]string{deviceDirectory, deviceFileName}, "/")
		deviceFileDownloading = fileUrl
		if err := DownloadFile(device.Name, deviceFullPath, fileUrl); err != nil {
			LogWarning(device.Name, fileUrl+" problem downloading "+err.Error())
		} else {
			DeleteFile(device.Name, fileUrl)
		}

	}
	if len(analogPorts) > 0 {
		LogInfo(device.Name, "Device has analog ports")
		fileUrl := "http://" + device.IpAddress + "/log/analog.txt"
		deviceDirectory := filepath.Join(".", strconv.Itoa(int(device.ID))+"-"+device.Name)
		deviceFileName := "analog.txt"
		deviceFullPath := strings.Join([]string{deviceDirectory, deviceFileName}, "/")
		deviceFileDownloading = fileUrl
		if err := DownloadFile(device.Name, deviceFullPath, fileUrl); err != nil {
			LogWarning(device.Name, fileUrl+" problem downloading "+err.Error())
		} else {
			DeleteFile(device.Name, fileUrl)
		}

	}
	if len(serialPorts) > 0 {
		LogInfo(device.Name, "Device has serial ports")
		fileUrl := "http://" + device.IpAddress + "/log/serial.txt"
		deviceDirectory := filepath.Join(".", strconv.Itoa(int(device.ID))+"-"+device.Name)
		deviceFileName := "serial.txt"
		deviceFullPath := strings.Join([]string{deviceDirectory, deviceFileName}, "/")
		deviceFileDownloading = fileUrl
		if err := DownloadFile(device.Name, deviceFullPath, fileUrl); err != nil {
			LogWarning(device.Name, fileUrl+" problem downloading "+err.Error())
		} else {
			DeleteFile(device.Name, fileUrl)
		}

	}
	if len(energyPorts) > 0 {
		LogInfo(device.Name, "Device has energy ports")
		fileUrl := "http://" + device.IpAddress + "/log/ui_value.txt"
		deviceDirectory := filepath.Join(".", strconv.Itoa(int(device.ID))+"-"+device.Name)
		deviceFileName := "ui_value.txt"
		deviceFullPath := strings.Join([]string{deviceDirectory, deviceFileName}, "/")
		deviceFileDownloading = fileUrl
		if err := DownloadFile(device.Name, deviceFullPath, fileUrl); err != nil {
			LogWarning(device.Name, fileUrl+" problem downloading "+err.Error())
		} else {
			DeleteFile(device.Name, fileUrl)
		}

	}
	deviceFileDownloading = ""
	defer db.Close()
	return true, nil
}

func DeleteFile(deviceName string, url string) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		LogError(deviceName, url+" cannot delete file : "+err.Error())
		return
	} else {
		resp, err := client.Do(req)
		if err != nil {
			LogError(deviceName, url+" cannot delete file: "+err.Error())
			return
		} else {
			LogInfo(deviceName, url+" file deleted")

		}
		defer resp.Body.Close()
	}
}

func DownloadFile(deviceName string, filepath string, url string) error {
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

func ProcessData(device zapsi_database.Device, intermediateData []IntermediateData) error {
	start := time.Now()
	connectionString, dialect := zapsi_database.CheckDatabaseType(DatabaseType, DatabaseIpAddress, DatabasePort, DatabaseLogin, DatabaseName, DatabasePassword)
	db, err := gorm.Open(dialect, connectionString)
	if err != nil {
		LogError(device.Name, "Problem opening "+DatabaseName+" database: "+err.Error())
		return err
	}
	defer db.Close()
	totalNumberOfRecords := len(intermediateData)
	displayProgress := true
	for progress, record := range intermediateData {
		switch record.Type {
		case digital:
			AddDigitalDataToDatabase(&record, db, device)
		case analog:
			AddAnalogDataToDatabase(&record, db, device)
		case serial:
			AddSerialDataToDatabase(&record, db, device)
		case energy:
			AddEnergyDataToDatabase(&record, db, device)
		}

		var virtualDigitalPorts []zapsi_database.DevicePort
		var virtualAnalogPorts []zapsi_database.DevicePort
		var virtualSerialPorts []zapsi_database.DevicePort
		var virtualEnergyPorts []zapsi_database.DevicePort
		db.Where("device_id = ?", device.ID).Where("device_port_type_id = ?", 1).Where("virtual = ?", true).Find(&virtualDigitalPorts)
		db.Where("device_id = ?", device.ID).Where("device_port_type_id = ?", 2).Where("virtual = ?", true).Find(&virtualAnalogPorts)
		db.Where("device_id = ?", device.ID).Where("device_port_type_id = ?", 3).Where("virtual = ?", true).Find(&virtualSerialPorts)
		db.Where("device_id = ?", device.ID).Where("device_port_type_id = ?", 4).Where("virtual = ?", true).Find(&virtualEnergyPorts)
		if len(virtualDigitalPorts) > 0 {
			AddVirtualDigitalDataToDatabase(record, virtualDigitalPorts, db, device)
		}
		if len(virtualAnalogPorts) > 0 {
			AddVirtualAnalogDataToDatabase(record, virtualAnalogPorts, db, device)
		}
		if len(virtualSerialPorts) > 0 {
			AddVirtualSerialDataToDatabase(record, virtualSerialPorts, db, device)
		}
		if len(virtualEnergyPorts) > 0 {
			AddVirtualEnergyDataToDatabase(record, virtualEnergyPorts, db, device)
		}

		if len(intermediateData) > 1000 {
			actualProgress := progress * 100 / totalNumberOfRecords
			if actualProgress != 0 && actualProgress%5 == 0 && displayProgress {
				displayProgress = false
				elapsedTime := time.Since(start)
				var remainingProgress = (100.00 - float64(actualProgress)) / (float64(actualProgress))
				var remainingTime = time.Duration(elapsedTime.Seconds() * remainingProgress * 1000000000)
				LogInfo(device.Name, strconv.Itoa(actualProgress)+"% done, remaining "+remainingTime.String())
			} else if actualProgress%5 != 0 {
				displayProgress = true
			}
		}
	}
	LogInfo(device.Name, "Processing takes "+time.Since(start).String())
	return nil
}

func AddVirtualEnergyDataToDatabase(record IntermediateData, virtualEnergyPorts []zapsi_database.DevicePort, db *gorm.DB, device zapsi_database.Device) {
	for _, virtualEnergyPort := range virtualEnergyPorts {
		result := ReplacePortNameWithItsValue(device, db, virtualEnergyPort.Settings)
		value, err := gval.Evaluate(result, nil)
		if err != nil {
			LogError(device.Name, "Problem evaluating data: "+err.Error())
			return
		}
		dateTimeToInsert := record.DateTime
		intervalToInsert := dateTimeToInsert.Sub(virtualEnergyPort.ActualDataDateTime)
		if intervalToInsert < 0 {
			LogWarning(device.Name, "Data for "+virtualEnergyPort.Name+" not inserting, data are older ["+dateTimeToInsert.String()+"] than data in database ["+virtualEnergyPort.ActualDataDateTime.String()+"]")
			return
		}

		recordToInsert := zapsi_database.DeviceAnalogRecord{DateTime: dateTimeToInsert, Data: float32(value.(float64)), DevicePortId: virtualEnergyPort.ID, Duration: intervalToInsert}
		db.NewRecord(recordToInsert)
		db.Create(&recordToInsert)
		virtualEnergyPort.ActualData = strconv.FormatFloat(value.(float64), 'g', 15, 64)
		virtualEnergyPort.ActualDataDateTime = dateTimeToInsert
		db.Save(&virtualEnergyPort)
	}
}

func AddVirtualSerialDataToDatabase(record IntermediateData, virtualSerialPorts []zapsi_database.DevicePort, db *gorm.DB, device zapsi_database.Device) {
	for _, virtualSerialPort := range virtualSerialPorts {
		result := ReplacePortNameWithItsValue(device, db, virtualSerialPort.Settings)
		value, err := gval.Evaluate(result, nil)
		if err != nil {
			LogError(device.Name, "Problem evaluating data: "+err.Error())
			return
		}
		dateTimeToInsert := record.DateTime
		intervalToInsert := dateTimeToInsert.Sub(virtualSerialPort.ActualDataDateTime)
		if intervalToInsert < 0 {
			LogWarning(device.Name, "Data for "+virtualSerialPort.Name+" not inserting, data are older ["+dateTimeToInsert.String()+"] than data in database ["+virtualSerialPort.ActualDataDateTime.String()+"]")
			return
		}
		recordToInsert := zapsi_database.DeviceSerialRecord{DateTime: dateTimeToInsert, Data: float32(value.(float64)), DevicePortId: virtualSerialPort.ID, Duration: intervalToInsert}
		db.NewRecord(recordToInsert)
		db.Create(&recordToInsert)
		virtualSerialPort.ActualData = strconv.FormatFloat(value.(float64), 'g', 15, 64)
		virtualSerialPort.ActualDataDateTime = dateTimeToInsert
		db.Save(&virtualSerialPort)
	}
}

func AddVirtualAnalogDataToDatabase(record IntermediateData, virtualAnalogPorts []zapsi_database.DevicePort, db *gorm.DB, device zapsi_database.Device) {
	for _, virtualAnalogPort := range virtualAnalogPorts {
		if strings.Contains(virtualAnalogPort.Settings, "SP:TC") {
			ProcessThermoCouplePort(record, virtualAnalogPort, db, device)
		} else if strings.Contains(virtualAnalogPort.Settings, "SP:SPEED") {
			ProcessSpeedPort(record, virtualAnalogPort, db, device)
		} else {
			ProcessDataAsStandardVirtualAnalogPort(record, virtualAnalogPort, db, device)
		}
	}
}

func ProcessThermoCouplePort(record IntermediateData, virtualPort zapsi_database.DevicePort, db *gorm.DB, device zapsi_database.Device) {
	parameters := strings.Split(virtualPort.Settings[6:len(virtualPort.Settings)-1], ";")
	thermoCoupleType := parameters[0]
	thermoCoupleMainPortId := parameters[1][1:]
	thermoCoupleColdJunctionPortId := parameters[2][1:]
	thermoCoupleTypeId := SelectThermoCouple(thermoCoupleType)
	ProcessThermoCouplePortData(record, thermoCoupleMainPortId, thermoCoupleColdJunctionPortId, thermoCoupleTypeId, virtualPort, db, device)
}

func ProcessThermoCouplePortData(record IntermediateData, thermoCoupleMainPortId string, thermoCoupleColdJunctionPortId string, thermoCoupleTypeId int, virtualPort zapsi_database.DevicePort, db *gorm.DB, device zapsi_database.Device) {
	var thermoCoupleMainPort zapsi_database.DevicePort
	var thermoCoupleColdJunctionPort zapsi_database.DevicePort
	db.Where("device_id = ?", device.ID).Where("port_number = ?", thermoCoupleMainPortId).Find(&thermoCoupleMainPort)
	db.Where("device_id = ?", device.ID).Where("port_number = ?", thermoCoupleColdJunctionPortId).Find(&thermoCoupleColdJunctionPort)
	thermoCoupleMainPortData, err := strconv.ParseFloat(thermoCoupleMainPort.ActualData, 64)
	if err != nil {
		LogError(device.Name, "Problem parsing data for thermocouple: "+err.Error())
		return
	}
	dataAsMv := math.Abs(thermoCoupleMainPortData) / 8.0 * 0.015625
	value := ConvertMvToTemp(dataAsMv, thermoCoupleTypeId)
	coldJunctionTemperature, err := strconv.ParseFloat(thermoCoupleColdJunctionPort.ActualData, 64)
	if err != nil {
		LogError(device.Name, "Problem parsing last data for coldjunction port, using 0: "+err.Error())
		coldJunctionTemperature = 0
	}
	value = value + coldJunctionTemperature
	dateTimeToInsert := record.DateTime
	intervalToInsert := dateTimeToInsert.Sub(virtualPort.ActualDataDateTime)
	if intervalToInsert < 0 {
		LogWarning(device.Name, "Data for "+virtualPort.Name+" not inserting, data are older ["+dateTimeToInsert.String()+"] than data in database ["+virtualPort.ActualDataDateTime.String()+"]")
		return
	}
	recordToInsert := zapsi_database.DeviceAnalogRecord{DateTime: dateTimeToInsert, Data: float32(value), DevicePortId: virtualPort.ID, Duration: intervalToInsert}
	db.NewRecord(recordToInsert)
	db.Create(&recordToInsert)
	virtualPort.ActualData = strconv.FormatFloat(value, 'g', 15, 64)
	virtualPort.ActualDataDateTime = dateTimeToInsert
	db.Save(&virtualPort)
}

func ProcessSpeedPort(record IntermediateData, virtualPort zapsi_database.DevicePort, db *gorm.DB, device zapsi_database.Device) {
	speed, err := CalculateSpeed(device, virtualPort, db)
	if err != nil {
		LogError(device.Name, "Problem evaluating data for speed port: "+err.Error())
		return
	}
	dateTimeToInsert := record.DateTime
	intervalToInsert := dateTimeToInsert.Sub(virtualPort.ActualDataDateTime)
	if intervalToInsert < 0 {
		LogWarning(device.Name, "Data for "+virtualPort.Name+" not inserting, data are older ["+dateTimeToInsert.String()+"] than data in database ["+virtualPort.ActualDataDateTime.String()+"]")
		return
	}
	recordToInsert := zapsi_database.DeviceAnalogRecord{DateTime: dateTimeToInsert, Data: float32(speed), DevicePortId: virtualPort.ID, Duration: intervalToInsert}
	db.NewRecord(recordToInsert)
	db.Create(&recordToInsert)
	virtualPort.ActualData = strconv.FormatFloat(speed, 'g', 15, 64)
	virtualPort.ActualDataDateTime = dateTimeToInsert
	db.Save(&virtualPort)
}

func CalculateSpeed(device zapsi_database.Device, virtualPort zapsi_database.DevicePort, db *gorm.DB) (float64, error) {
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
	var devicePort zapsi_database.DevicePort
	db.Where("device_id = ?", device.ID).Where("port_number = ?", portNumber).Find(&devicePort)
	var digitalRecords []zapsi_database.DeviceDigitalRecord
	db.Where("device_port_id = ?", devicePort.ID).Where("date_time > ?", timeForData).Where("data = ?", 0).Find(&digitalRecords)
	speed := float64(len(digitalRecords)) * diameter * math.Pi
	return speed, nil
}

func ProcessDataAsStandardVirtualAnalogPort(record IntermediateData, virtualPort zapsi_database.DevicePort, db *gorm.DB, device zapsi_database.Device) {
	result := ReplacePortNameWithItsValue(device, db, virtualPort.Settings)
	value, err := gval.Evaluate(result, nil)
	if err != nil {
		LogError(device.Name, "Problem evaluating data: "+err.Error())
		return
	}
	dateTimeToInsert := record.DateTime
	intervalToInsert := dateTimeToInsert.Sub(virtualPort.ActualDataDateTime)
	if intervalToInsert < 0 {
		LogWarning(device.Name, "Data for "+virtualPort.Name+" not inserting, data are older ["+dateTimeToInsert.String()+"] than data in database ["+virtualPort.ActualDataDateTime.String()+"]")
		return
	}
	recordToInsert := zapsi_database.DeviceAnalogRecord{DateTime: dateTimeToInsert, Data: float32(value.(float64)), DevicePortId: virtualPort.ID, Duration: intervalToInsert}
	db.NewRecord(recordToInsert)
	db.Create(&recordToInsert)
	virtualPort.ActualData = strconv.FormatFloat(value.(float64), 'g', 15, 64)
	virtualPort.ActualDataDateTime = dateTimeToInsert
	db.Save(&virtualPort)

}

func AddVirtualDigitalDataToDatabase(record IntermediateData, virtualDigitalPorts []zapsi_database.DevicePort, db *gorm.DB, device zapsi_database.Device) {
	for _, virtualDigitalPort := range virtualDigitalPorts {
		if strings.Contains(virtualDigitalPort.Settings, "SP:ADDZERO") {
			ProcessDataAsAddZeroPort(record, virtualDigitalPort, db, device)
		} else {
			ProcessDataAsStandardVirtualDigitalPort(record, virtualDigitalPort, db, device)
		}
	}

}

func ProcessDataAsAddZeroPort(data IntermediateData, virtualPort zapsi_database.DevicePort, db *gorm.DB, device zapsi_database.Device) {
	if data.Type == digital {
		originalPort := virtualPort.Settings[12 : len(virtualPort.Settings)-1]
		originalPortId, err := strconv.ParseUint(originalPort, 10, 64)
		if err != nil {
			LogError(device.Name, "Problem parsing settings from port "+virtualPort.Name+" ["+virtualPort.Settings+"]: "+err.Error())
			return
		}
		originalPortIdUint := uint(originalPortId)
		var digitalPorts []zapsi_database.DevicePort
		db.Where("device_id = ?", device.ID).Where("device_port_type_id = ?", 1).Where("virtual = ?", false).Find(&digitalPorts)
		for _, port := range digitalPorts {
			if port.ID == originalPortIdUint {
				db.LogMode(false)
				positionInFile := port.PortNumber - 1
				parsedData := strings.Split(data.RawData, ";")
				dataToInsert, err := strconv.Atoi(parsedData[positionInFile])
				if err != nil {
					LogError(device.Name, "Problem parsing data: "+err.Error())
				}
				if dataToInsert == 1 {
					firstDateTimeToInsert := data.DateTime
					secondDateTimeToInsert := data.DateTime.Add(1 * time.Second)
					firstIntervalToInsert := firstDateTimeToInsert.Sub(virtualPort.ActualDataDateTime)
					secondIntervalToInsert := time.Second * 1
					if firstIntervalToInsert < 0 {
						LogWarning(device.Name, "Data for "+port.Name+" not inserting, data are older ["+firstDateTimeToInsert.String()+"] than data in database ["+port.ActualDataDateTime.String()+"]")
						continue
					}
					ActualData, err := strconv.Atoi(virtualPort.ActualData)
					if err != nil {
						ActualData = 0
					}
					if ActualData != dataToInsert {
						firstRecord := zapsi_database.DeviceDigitalRecord{DateTime: firstDateTimeToInsert, Data: dataToInsert, DevicePortId: virtualPort.ID, Duration: firstIntervalToInsert}
						secondRecord := zapsi_database.DeviceDigitalRecord{DateTime: secondDateTimeToInsert, Data: 0, DevicePortId: virtualPort.ID, Duration: secondIntervalToInsert}
						db.NewRecord(firstRecord)
						db.NewRecord(secondRecord)
						db.Create(&firstRecord)
						db.Create(&secondRecord)
						virtualPort.ActualData = "0"
						virtualPort.ActualDataDateTime = secondDateTimeToInsert
						db.Save(&virtualPort)
					} else {
						LogWarning(device.Name, "Data mismatch for "+port.Name+": last data is equal with new data: ["+strconv.Itoa(ActualData)+";"+strconv.Itoa(dataToInsert)+"]")
					}
				}
			}
		}
	}
}

func ProcessDataAsStandardVirtualDigitalPort(record IntermediateData, virtualPort zapsi_database.DevicePort, db *gorm.DB, device zapsi_database.Device) {
	result := ReplacePortNameWithItsValue(device, db, virtualPort.Settings)
	value, err := gval.Evaluate(result, nil)
	if err != nil {
		LogError(device.Name, "Problem evaluating data: "+err.Error())
		return
	}
	dataToInsert := 0
	if value.(bool) == true {
		dataToInsert = 1
	}
	dateTimeToInsert := record.DateTime
	intervalToInsert := dateTimeToInsert.Sub(virtualPort.ActualDataDateTime)
	if intervalToInsert < 0 {
		LogWarning(device.Name, "Data for "+virtualPort.Name+" not inserting, data are older ["+dateTimeToInsert.String()+"] than data in database ["+virtualPort.ActualDataDateTime.String()+"]")
		return
	}

	ActualData, err := strconv.Atoi(virtualPort.ActualData)
	if err != nil {
		ActualData = 0
	}
	if ActualData != dataToInsert {
		recordToInsert := zapsi_database.DeviceDigitalRecord{DateTime: dateTimeToInsert, Data: dataToInsert, DevicePortId: virtualPort.ID, Duration: intervalToInsert}
		db.NewRecord(recordToInsert)
		db.Create(&recordToInsert)
		virtualPort.ActualData = strconv.Itoa(dataToInsert)
		virtualPort.ActualDataDateTime = dateTimeToInsert
		db.Save(&virtualPort)
	} else {
		LogWarning(device.Name, "Data mismatch for "+virtualPort.Name+": last data is equal with new data: ["+strconv.Itoa(ActualData)+";"+strconv.Itoa(dataToInsert)+"]")
	}
}

func ReplacePortNameWithItsValue(device zapsi_database.Device, db *gorm.DB, settings string) string {
	var digitalPorts []zapsi_database.DevicePort
	var analogPorts []zapsi_database.DevicePort
	var serialPorts []zapsi_database.DevicePort
	var energyPorts []zapsi_database.DevicePort
	db.Where("device_id = ?", device.ID).Where("device_port_type_id = ?", 1).Where("virtual = ?", false).Find(&digitalPorts)
	db.Where("device_id = ?", device.ID).Where("device_port_type_id = ?", 2).Where("virtual = ?", false).Find(&analogPorts)
	db.Where("device_id = ?", device.ID).Where("device_port_type_id = ?", 3).Where("virtual = ?", false).Find(&serialPorts)
	db.Where("device_id = ?", device.ID).Where("device_port_type_id = ?", 4).Where("virtual = ?", false).Find(&energyPorts)
	for _, digitalPort := range digitalPorts {
		settings = ReplacePortWithItsValue("D", settings, digitalPort)

	}
	for _, analogPort := range analogPorts {
		settings = ReplacePortWithItsValue("A", settings, analogPort)

	}
	for _, serialPort := range serialPorts {
		settings = ReplacePortWithItsValue("S", settings, serialPort)

	}
	for _, energyPort := range energyPorts {
		settings = ReplacePortWithItsValue("E", settings, energyPort)

	}
	return settings
}

func ReplacePortWithItsValue(portType string, settings string, port zapsi_database.DevicePort) string {
	if strings.Contains(settings, portType+strconv.Itoa(port.PortNumber)) {
		return strings.Replace(settings, portType+strconv.Itoa(port.PortNumber), port.ActualData, -1)
	}
	return settings
}

func PrepareData(device zapsi_database.Device) []IntermediateData {
	var intermediateData []IntermediateData
	if FileExists("digital.txt", device) {
		AddDataForProcessing("digital.txt", &intermediateData, device)
	}
	if FileExists("analog.txt", device) {
		AddDataForProcessing("analog.txt", &intermediateData, device)
	}
	if FileExists("serial.txt", device) {
		AddDataForProcessing("serial.txt", &intermediateData, device)
	}
	if FileExists("ui_value.txt", device) {
		AddDataForProcessing("ui_value.txt", &intermediateData, device)
	}
	sort.Slice(intermediateData, func(i, j int) bool {
		return intermediateData[i].DateTime.Before(intermediateData[j].DateTime)
	})
	LogInfo(device.Name, "Data sorted, number of records: "+strconv.Itoa(len(intermediateData)))
	return intermediateData
}

func AddEnergyDataToDatabase(data *IntermediateData, db *gorm.DB, device zapsi_database.Device) {
	var energyPorts []zapsi_database.DevicePort
	db.Where("device_id = ?", device.ID).Where("device_port_type_id = ?", 4).Where("virtual = ?", false).Find(&energyPorts)
	for _, port := range energyPorts {
		db.LogMode(false)
		positionInFile := port.PortNumber - 1
		parsedData := strings.Split(data.RawData, ";")
		dataToInsert, err := strconv.ParseFloat(parsedData[positionInFile], 32)
		if err != nil {
			LogError(device.Name, "Problem parsing data: "+err.Error())
		}
		dateTimeToInsert := data.DateTime
		intervalToInsert := dateTimeToInsert.Sub(port.ActualDataDateTime)
		if intervalToInsert < 0 {
			LogWarning(device.Name, "Data for "+port.Name+" not inserting, data are older ["+dateTimeToInsert.String()+"] than data in database ["+port.ActualDataDateTime.String()+"]")
			continue
		}
		recordToInsert := zapsi_database.DeviceAnalogRecord{DateTime: dateTimeToInsert, Data: float32(dataToInsert), DevicePortId: port.ID, Duration: intervalToInsert}
		db.NewRecord(recordToInsert)
		db.Create(&recordToInsert)

		port.ActualData = parsedData[positionInFile]
		port.ActualDataDateTime = dateTimeToInsert
		db.Save(&port)
	}
}

func AddSerialDataToDatabase(data *IntermediateData, db *gorm.DB, device zapsi_database.Device) {
	var serialPorts []zapsi_database.DevicePort
	db.Where("device_id = ?", device.ID).Where("device_port_type_id = ?", 3).Where("virtual = ?", false).Find(&serialPorts)
	for _, port := range serialPorts {
		db.LogMode(false)
		positionInFile := port.PortNumber - 1
		parsedData := strings.Split(data.RawData, ";")
		dataToInsert, err := strconv.ParseFloat(parsedData[positionInFile], 32)
		if err != nil {
			LogError(device.Name, "Problem parsing data: "+err.Error())
		}
		dateTimeToInsert := data.DateTime
		intervalToInsert := dateTimeToInsert.Sub(port.ActualDataDateTime)
		if intervalToInsert < 0 {
			LogWarning(device.Name, "Data for "+port.Name+" not inserting, data are older ["+dateTimeToInsert.String()+"] than data in database ["+port.ActualDataDateTime.String()+"]")
			continue
		}
		recordToInsert := zapsi_database.DeviceSerialRecord{DateTime: dateTimeToInsert, Data: float32(dataToInsert), DevicePortId: port.ID, Duration: intervalToInsert}
		db.NewRecord(recordToInsert)
		db.Create(&recordToInsert)
		port.ActualData = parsedData[positionInFile]
		port.ActualDataDateTime = dateTimeToInsert
		db.Save(&port)
	}
}

func AddDigitalDataToDatabase(data *IntermediateData, db *gorm.DB, device zapsi_database.Device) {
	var digitalPorts []zapsi_database.DevicePort
	db.Where("device_id = ?", device.ID).Where("device_port_type_id = ?", 1).Where("virtual = ?", false).Find(&digitalPorts)
	for _, port := range digitalPorts {
		db.LogMode(false)
		positionInFile := port.PortNumber - 1
		parsedData := strings.Split(data.RawData, ";")
		dataToInsert, err := strconv.Atoi(parsedData[positionInFile])
		if err != nil {
			LogError(device.Name, "Problem parsing data: "+err.Error())
		}
		dateTimeToInsert := data.DateTime
		intervalToInsert := dateTimeToInsert.Sub(port.ActualDataDateTime)
		if intervalToInsert < 0 {
			LogWarning(device.Name, "Data for "+port.Name+" not inserting, data are older ["+dateTimeToInsert.String()+"] than data in database ["+port.ActualDataDateTime.String()+"]")
			continue
		}
		ActualData, err := strconv.Atoi(port.ActualData)
		if err != nil {
			ActualData = 0
		}
		if ActualData != dataToInsert {
			recordToInsert := zapsi_database.DeviceDigitalRecord{DateTime: dateTimeToInsert, Data: dataToInsert, DevicePortId: port.ID, Duration: intervalToInsert}
			db.NewRecord(recordToInsert)
			db.Create(&recordToInsert)
			port.ActualData = parsedData[positionInFile]
			port.ActualDataDateTime = dateTimeToInsert
			db.Save(&port)
		} else {
			LogWarning(device.Name, "Data mismatch for "+port.Name+": last data is equal with new data: ["+strconv.Itoa(ActualData)+";"+strconv.Itoa(dataToInsert)+"]")
		}
	}
}

func AddAnalogDataToDatabase(data *IntermediateData, db *gorm.DB, device zapsi_database.Device) {
	var analogPorts []zapsi_database.DevicePort
	db.Where("device_id = ?", device.ID).Where("device_port_type_id = ?", 2).Where("virtual = ?", false).Find(&analogPorts)
	for _, port := range analogPorts {
		db.LogMode(false)
		positionInFile := port.PortNumber - 1
		parsedData := strings.Split(data.RawData, ";")
		dataToInsert, err := strconv.ParseFloat(parsedData[positionInFile], 32)
		if err != nil {
			LogError(device.Name, "Problem parsing data: "+err.Error())
		}
		dateTimeToInsert := data.DateTime
		intervalToInsert := dateTimeToInsert.Sub(port.ActualDataDateTime)
		if intervalToInsert < 0 {
			LogWarning(device.Name, "Data for "+port.Name+" not inserting, data are older ["+dateTimeToInsert.String()+"] than data in database ["+port.ActualDataDateTime.String()+"]")
			continue
		}
		recordToInsert := zapsi_database.DeviceAnalogRecord{DateTime: dateTimeToInsert, Data: float32(dataToInsert), DevicePortId: port.ID, Duration: intervalToInsert}
		db.NewRecord(recordToInsert)
		db.Create(&recordToInsert)

		port.ActualData = parsedData[positionInFile]
		port.ActualDataDateTime = dateTimeToInsert
		db.Save(&port)
	}
}

func FileExists(filename string, device zapsi_database.Device) bool {
	deviceDirectory := filepath.Join(".", strconv.Itoa(int(device.ID))+"-"+device.Name)
	deviceFullPath := strings.Join([]string{deviceDirectory, filename}, "/")
	if _, err := os.Stat(deviceFullPath); err == nil {
		return true
	} else if os.IsNotExist(err) {
		return false
	} else {
		return false
	}
}

func AddDataForProcessing(filename string, data *[]IntermediateData, device zapsi_database.Device) {
	deviceDirectory := filepath.Join(".", strconv.Itoa(int(device.ID))+"-"+device.Name)
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
		finalDateTime, err := GetDateTimeFromData(parsedData)
		if err != nil {
			LogError(device.Name, "Problem parsing datetime from ["+zapsiData+"], "+err.Error())
			continue
		}
		AddIntermediateData(finalDateTime, rawData, filename, data)
	}
}

func AddIntermediateData(finalDateTime time.Time, rawData string, filename string, data *[]IntermediateData) {
	dataForInsert := IntermediateData{DateTime: finalDateTime, RawData: rawData}
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

func GetDateTimeFromData(data []string) (time.Time, error) {
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
		input := dataYear + "-" + dataMonth + "-" + dataDay + " " + dataHour + ":" + dataMinute + ":" + dataSecond + "." + dataMilliSecond
		var layout string
		switch len(dataMilliSecond) {
		case 1:
			layout = "2006-1-2 15:4:5.0"
		case 2:
			layout = "2006-1-2 15:4:5.00"
		default:
			layout = "2006-1-2 15:4:5.000"
		}

		finalDateTime, err := time.Parse(layout, input)
		return finalDateTime, err
	}
	return time.Now(), BadDataError{}
}

func (e BadDataError) Error() string {
	return fmt.Sprintf("bad line in input data")
}

func SendUDP(device zapsi_database.Device, dstIP string, dstPort int, localIP string, localPort uint, data []byte) {
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
func SendTime(device zapsi_database.Device) (timeUpdated bool) {
	dateTimeForZapsi := time.Now().UTC().Format("02.01.2006 15:04:05")
	dateTimeForZapsi = "set_datetime=" + dateTimeForZapsi + " 0" + strconv.Itoa(int(time.Now().UTC().Weekday())) + "&"
	SendUDP(device, device.IpAddress, 9999, "", 0, []byte(dateTimeForZapsi))
	return true
}

func SendTimeToZapsi(device zapsi_database.Device, timeUpdated bool) bool {
	now := time.Now().UTC()
	dateTimeForZapsi := now.Format("02.01.2006 15:04:05")

	if now.Hour() == setZapsiTimeAtHour && now.Minute() == setZapsiTimeAtMinute && !timeUpdated {
		dateTimeForZapsi = "set_datetime=" + dateTimeForZapsi + " 0" + strconv.Itoa(int(now.Weekday())) + "&"
		SendUDP(device, device.IpAddress, 9999, "", 0, []byte(dateTimeForZapsi))
		return true
	}
	if now.Hour() == setZapsiTimeAtHour && now.Minute() == setZapsiTimeAtMinute && timeUpdated {
		return true
	}
	return false
}
