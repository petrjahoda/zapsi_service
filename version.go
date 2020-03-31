package main

import (
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

func Version() {
	readFile, err := ioutil.ReadFile("create.sh")
	if err != nil {
		println("Error reading create.sh")
		os.Exit(1)
	}
	println("File opened")

	result := ""
	for _, line := range strings.Split(strings.TrimSuffix(string(readFile), "\n"), "\n") {
		if strings.Contains(line, ":2") {
			date := time.Now()
			day := date.Day()
			year := date.Year()
			month := date.Month()
			quarter := (month-1)/3 + 1
			monthInQuarter := month - 3*(quarter-1)
			result += "const version = \"" + strconv.Itoa(year) + "." + strconv.Itoa(int(quarter)) + "." + strconv.Itoa(int(monthInQuarter)) + "." + strconv.Itoa(day) + "\"\n"
		} else {
			result += line + "\n"
		}
	}
	err = ioutil.WriteFile("create.sh", []byte(result), 0644)
	if err != nil {
		println("Error writing create.sh")
		os.Exit(1)
	}
	println("Updated")

}
