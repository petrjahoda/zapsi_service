package main

import (
	"time"
)

type IntermediateData struct {
	Type     IntermediateDataType
	DateTime time.Time
	RawData  string
}

type IntermediateDataType int

const (
	digital IntermediateDataType = iota
	analog
	serial
	energy
)
