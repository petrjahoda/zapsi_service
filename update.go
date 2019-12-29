package main

import (
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

func Update() {
	readFile, err := ioutil.ReadFile("main.go")
	if err != nil {
		println("Error reading main.go")
		os.Exit(1)
	}
	println("File opened")

	result := ""
	for _, line := range strings.Split(strings.TrimSuffix(string(readFile), "\n"), "\n") {
		if strings.Contains(line, "const version =") {
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
	err = ioutil.WriteFile("main.go", []byte(result), 0644)
	if err != nil {
		println("Error writing main.go")
		os.Exit(1)
	}
	println("Updated")

}
