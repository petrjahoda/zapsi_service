package main

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"time"
)

type DeviceType struct {
	gorm.Model
	Name string `gorm:"unique"`
	Note string
}

type DevicePortType struct {
	gorm.Model
	Name string `gorm:"unique"`
	Note string
}

type Setting struct {
	gorm.Model
	Key     string `gorm:"unique"`
	Value   string
	Enabled bool
	Note    string
}

type Device struct {
	gorm.Model
	Name        string `gorm:"unique"`
	DeviceType  uint
	IpAddress   string `gorm:"unique"`
	MacAddress  string
	TypeName    string
	IsActivated bool
	Settings    string
	Workplace   uint
	DevicePorts []DevicePort
	Note        string
}

type DevicePort struct {
	gorm.Model
	Name               string
	Unit               string
	PortNumber         int
	DevicePortTypeId   uint
	DeviceId           uint
	ActualDataDateTime time.Time
	ActualData         string
	PlcDataType        string
	PlcDataAddress     string
	Settings           string
	Virtual            bool
	Note               string
}

type DeviceAnalogRecord struct {
	Id           uint      `gorm:"primary_key"`
	DevicePortId uint      `gorm:"unique_index:unique_analog_data"`
	DateTime     time.Time `gorm:"unique_index:unique_analog_data"`
	Data         float32
	Interval     float32
}

type DeviceDigitalRecord struct {
	Id           uint      `gorm:"primary_key"`
	DevicePortId uint      `gorm:"unique_index:unique_digital_data"`
	DateTime     time.Time `gorm:"unique_index:unique_digital_data"`
	Data         int
	Interval     float32
}

type DeviceSerialRecord struct {
	Id           uint      `gorm:"primary_key"`
	DevicePortId uint      `gorm:"unique_index:unique_serial_data"`
	DateTime     time.Time `gorm:"unique_index:unique_serial_data"`
	Data         float32
	Interval     float32
}

func CheckDatabase() {
	var connectionString string
	var defaultString string
	var dialect string
	if DatabaseType == "postgres" {
		connectionString = "host=" + DatabaseIpAddress + " sslmode=disable port=" + DatabasePort + " user=" + DatabaseLogin + " dbname=" + DatabaseName + " password=" + DatabasePassword
		defaultString = "host=" + DatabaseIpAddress + " sslmode=disable port=" + DatabasePort + " user=" + DatabaseLogin + " dbname=postgres password=" + DatabasePassword
		dialect = "postgres"
	} else if DatabaseType == "mysql" {
		connectionString = DatabaseLogin + ":" + DatabasePassword + "@tcp(" + DatabaseIpAddress + ":" + DatabasePort + ")/" + DatabaseName + "?charset=utf8&parseTime=True&loc=Local"
		defaultString = DatabaseLogin + ":" + DatabasePassword + "@tcp(" + DatabaseIpAddress + ":" + DatabasePort + ")/information_schema?charset=utf8&parseTime=True&loc=Local"
		dialect = "mysql"
	}
	db, err := gorm.Open(dialect, connectionString)
	if err != nil {
		LogWarning("MAIN", "Database zapsi4 does not exist")
		db, err = gorm.Open(dialect, defaultString)
		if err != nil {
			LogError("MAIN", "Problem opening postgres database: "+err.Error())
			return
		}
		db = db.Exec("CREATE DATABASE zapsi4;")
		if db.Error != nil {
			LogError("MAIN", "Cannot create database zapsi4")
		}
		LogInfo("MAIN", "Database zapsi4 created")

	}
	defer db.Close()
	LogDebug("MAIN", "Database zapsi4 exists")
}

func CheckTables() {
	connectionString, dialect := CheckDatabaseType()
	db, err := gorm.Open(dialect, connectionString)
	if err != nil {
		LogError("MAIN", "Problem opening "+dialect+" database: "+err.Error())
		return
	}
	defer db.Close()
	if !db.HasTable(&DeviceType{}) {
		LogInfo("MAIN", "DeviceType table not exists, creating")
		db.CreateTable(&DeviceType{})
		zapsi := DeviceType{Name: "Zapsi"}
		db.NewRecord(zapsi)
		db.Create(&zapsi)
		zapsiTouchOriginal := DeviceType{Name: "Zapsi Touch Original"}
		db.NewRecord(zapsiTouchOriginal)
		db.Create(&zapsiTouchOriginal)
		zapsiTouchVirtual := DeviceType{Name: "Zapsi Touch Virtual"}
		db.NewRecord(zapsiTouchVirtual)
		db.Create(&zapsiTouchVirtual)
		zapsiTouchRpiOne := DeviceType{Name: "Zapsi Touch Rpi 1"}
		db.NewRecord(zapsiTouchRpiOne)
		db.Create(&zapsiTouchRpiOne)
		zapsiTouchRpiTwo := DeviceType{Name: "Zapsi Touch Rpi 2"}
		db.NewRecord(zapsiTouchRpiTwo)
		db.Create(&zapsiTouchRpiTwo)
		siemens := DeviceType{Name: "Siemens"}
		db.NewRecord(siemens)
		db.Create(&siemens)
		opc := DeviceType{Name: "OPC"}
		db.NewRecord(opc)
		db.Create(&opc)
		scale := DeviceType{Name: "Scale"}
		db.NewRecord(scale)
		db.Create(&scale)
		printer := DeviceType{Name: "Printer"}
		db.NewRecord(printer)
		db.Create(&printer)
		fileImport := DeviceType{Name: "File Import"}
		db.NewRecord(fileImport)
		db.Create(&fileImport)
		smtp := DeviceType{Name: "SMTP"}
		db.NewRecord(smtp)
		db.Create(&smtp)
	} else {
		db.AutoMigrate(&DeviceType{})
	}
	if !db.HasTable(&Device{}) {
		LogInfo("MAIN", "Device table not exists, creating")
		db.CreateTable(&Device{})
		db.Model(&Device{}).AddForeignKey("device_type_id", "device_types(id)", "RESTRICT", "RESTRICT")
	} else {
		db.AutoMigrate(&Device{})
	}
	if !db.HasTable(&Setting{}) {
		LogInfo("MAIN", "Setting table not exists, creating")
		db.CreateTable(&Setting{})
		host := Setting{Key: "host", Value: "smtp.forpsi.com"}
		db.NewRecord(host)
		db.Create(&host)
		port := Setting{Key: "port", Value: "587"}
		db.NewRecord(port)
		db.Create(&port)
		username := Setting{Key: "username", Value: "jahoda@zapsi.eu"}
		db.NewRecord(username)
		db.Create(&username)
		password := Setting{Key: "password", Value: "password"}
		db.NewRecord(password)
		db.Create(&password)
		email := Setting{Key: "email", Value: "support@zapsi.eu"}
		db.NewRecord(email)
		db.Create(&email)
	} else {
		db.AutoMigrate(&Setting{})
	}
	if !db.HasTable(&DevicePortType{}) {
		LogInfo("MAIN", "DevicePortType table not exists, creating")
		db.CreateTable(&DevicePortType{})
		digital := DevicePortType{Name: "Digital"}
		db.NewRecord(digital)
		db.Create(&digital)
		analog := DevicePortType{Name: "Analog"}
		db.NewRecord(analog)
		db.Create(&analog)
		serial := DevicePortType{Name: "Serial"}
		db.NewRecord(serial)
		db.Create(&serial)
		special := DevicePortType{Name: "Special"}
		db.NewRecord(special)
		db.Create(&special)
	} else {
		db.AutoMigrate(&DevicePortType{})
	}
	if !db.HasTable(&DevicePort{}) {
		LogInfo("MAIN", "DevicePort table not exists, creating")
		db.CreateTable(&DevicePort{})
		db.Model(&DevicePort{}).AddForeignKey("device_id", "devices(id)", "RESTRICT", "RESTRICT")
		db.Model(&DevicePort{}).AddForeignKey("device_port_type_id", "device_port_types(id)", "RESTRICT", "RESTRICT")
	} else {
		db.AutoMigrate(&DevicePort{})
	}
	if !db.HasTable(&DeviceAnalogRecord{}) {
		LogInfo("MAIN", "DeviceAnalogRecord table not exists, creating")
		db.CreateTable(&DeviceAnalogRecord{})
		db.Model(&DeviceAnalogRecord{}).AddForeignKey("device_port_id", "device_ports(id)", "RESTRICT", "RESTRICT")
	} else {
		db.AutoMigrate(&DeviceAnalogRecord{})
	}
	if !db.HasTable(&DeviceDigitalRecord{}) {
		LogInfo("MAIN", "DeviceDigitalRecord table not exists, creating")
		db.CreateTable(&DeviceDigitalRecord{})
		db.Model(&DeviceDigitalRecord{}).AddForeignKey("device_port_id", "device_ports(id)", "RESTRICT", "RESTRICT")
	} else {
		db.AutoMigrate(&DeviceDigitalRecord{})
	}
	if !db.HasTable(&DeviceSerialRecord{}) {
		LogInfo("MAIN", "DeviceSerialRecord table not exists, creating")
		db.CreateTable(&DeviceSerialRecord{})
		db.Model(&DeviceSerialRecord{}).AddForeignKey("device_port_id", "device_ports(id)", "RESTRICT", "RESTRICT")
	} else {
		db.AutoMigrate(&DeviceSerialRecord{})
	}

}

func CheckDatabaseType() (string, string) {
	var connectionString string
	var dialect string
	if DatabaseType == "postgres" {
		connectionString = "host=" + DatabaseIpAddress + " sslmode=disable port=" + DatabasePort + " user=" + DatabaseLogin + " dbname=" + DatabaseName + " password=" + DatabasePassword
		dialect = "postgres"
	} else if DatabaseType == "mysql" {
		connectionString = DatabaseLogin + ":" + DatabasePassword + "@tcp(" + DatabaseIpAddress + ":" + DatabasePort + ")/" + DatabaseName + "?charset=utf8&parseTime=True&loc=Local"
		dialect = "mysql"
	}
	return connectionString, dialect
}
