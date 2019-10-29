package main

import (
	"github.com/jinzhu/gorm"
	"gopkg.in/gomail.v2"
	"os"
	"strconv"
)

func SendMail(subject string, message string) {
	err, host, port, username, password, email := UpdateMailSettings()
	if err != nil {
		return
	}
	name, err := os.Hostname()
	if err != nil {
		LogError("MAIN", "Problem getting name of the computer, "+err.Error())
		name = ""
	}
	m := gomail.NewMessage()
	m.SetHeader("From", username)
	m.SetHeader("To", email)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", name+": "+message)
	d := gomail.NewDialer(host, port, username, password) // PETRzpsMAIL79..
	if emailSentError := d.DialAndSend(m); emailSentError != nil {
		LogError("MAIN", "Email not sent: "+emailSentError.Error())
	} else {
		LogInfo("MAIN", "Email sent: "+subject)
	}
}

func UpdateMailSettings() (error, string, int, string, string, string) {
	connectionString, dialect := CheckDatabaseType()
	db, err := gorm.Open(dialect, connectionString)
	if err != nil {
		LogError("MAIN", "Problem opening "+DatabaseName+" database: "+err.Error())
		return nil, "", 0, "", "", ""
	}
	var settingsHost Setting
	db.Where("Key=?", "host").Find(&settingsHost)
	host := settingsHost.Value
	var settingsPort Setting
	db.Where("Key=?", "port").Find(&settingsPort)
	port, err := strconv.Atoi(settingsPort.Value)
	if err != nil {
		LogError("MAIN", "Problem parsing port for email, using default port 587 "+err.Error())
		port = 587
	}
	var settingsUsername Setting
	db.Where("Key=?", "username").Find(&settingsUsername)
	username := settingsUsername.Value
	var settingsPassword Setting
	db.Where("Key=?", "password").Find(&settingsPassword)
	password := settingsPassword.Value
	var settingsEmail Setting
	db.Where("Key=?", "email").Find(&settingsEmail)
	email := settingsEmail.Value
	LogDebug("MAIN", "Mail settings: "+host+":"+strconv.Itoa(port)+" ["+username+"] ["+password+"]")
	defer db.Close()
	return err, host, port, username, password, email
}
