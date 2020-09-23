[![developed_using](https://img.shields.io/badge/developed%20using-Jetbrains%20Goland-lightgrey)](https://www.jetbrains.com/go/)
<br/>
![GitHub](https://img.shields.io/github/license/petrjahoda/zapsi_service)
[![GitHub last commit](https://img.shields.io/github/last-commit/petrjahoda/zapsi_service)](https://github.com/petrjahoda/zapsi_service/commits/master)
[![GitHub issues](https://img.shields.io/github/issues/petrjahoda/zapsi_service)](https://github.com/petrjahoda/zapsi_service/issues)
<br/>
![GitHub language count](https://img.shields.io/github/languages/count/petrjahoda/zapsi_service)
![GitHub top language](https://img.shields.io/github/languages/top/petrjahoda/zapsi_service)
![GitHub repo size](https://img.shields.io/github/repo-size/petrjahoda/zapsi_service)
<br/>
[![Docker Pulls](https://img.shields.io/docker/pulls/petrjahoda/zapsi_service)](https://hub.docker.com/r/petrjahoda/zapsi_service)
[![Docker Image Size (latest by date)](https://img.shields.io/docker/image-size/petrjahoda/zapsi_service?sort=date)](https://hub.docker.com/r/petrjahoda/zapsi_service/tags)
<br/>
[![developed_using](https://img.shields.io/badge/database-PostgreSQL-red)](https://www.postgresql.org) [![developed_using](https://img.shields.io/badge/runtime-Docker-red)](https://www.docker.com)

# Zapsi Service

## Description
Go service downloads data from [zapsi devices](https://www.zapsi.eu).

## Installation Information
Install under docker runtime using [this dockerfile image](https://github.com/petrjahoda/system/tree/master/latest) with this command: ```docker-compose up -d```

## Implementation Information
Check the software running with this command: ```docker stats```. <br/>
Zapsi_service has to be running.

### Setup device for download
1. Add device to database: insert new data into ```devices``` table
    - insert device name into ```name``` column
    - insert 1 into ```device_type_id``` column
    - set ```activated``` column to true
    - insert proper ip address to ```ip_address``` column
    - other columns are optional
2. Setup data for download: check data on device in /log directory
    - for analog.txt: insert new data into ```device_ports``` table
        - insert port name into ```name``` column
        - insert proper device_id into ```device_id``` column
        - insert 2 into ```device_port_type_id``` column
        - insert position from analog.txt into ```port_number``` column (1 means first position in analog.txt, 2 means second position in analog.txt, etc)
        - insert unit into ```unit``` column
        - set ```virtual``` column to false
    - for digital.txt: insert new data into ```device_ports``` table
        - insert port name into ```name``` column
        - insert proper device_id into ```device_id``` column
        - insert 1 into ```device_port_type_id``` column
        - insert position from digital.txt into ```port_number``` column (1 means first position in digital.txt, 2 means second position in digital.txt, etc)
        - insert unit into ```unit``` column
        - set ```virtual``` column to false 
    - for serial.txt: insert new data into ```device_ports``` table
        - insert port name into ```name``` column
        - insert proper device_id into ```device_id``` column
        - insert 3 into ```device_port_type_id``` column
        - insert position from serial.txt into ```port_number``` column (1 means first position in serial.txt, 2 means second position in serial.txt, etc)
        - insert unit into ```unit``` column
        - set ```virtual``` column to false
    - for ui_value.txt: insert new data into ```device_ports``` table
        - insert port name into ```name``` column
        - insert proper device_id into ```device_id``` column
        - insert 4 into ```device_port_type_id``` column
        - insert position from ui_value.txt into ```port_number``` column (1 means first position in ui_value.txt, 2 means second position in ui_value.txt, etc)
        - insert unit into ```unit``` column
        - set ```virtual``` column to false

3. Setup virtual ports (additional)
Virtual ports are calculated from physical data on-the-fly. That means data are inserted into database in exactly the same time as original data from device.
To add virtual port
    - insert all data into ```device_ports``` table as above
    - set ```port_number``` to specific value not found in physical files (use 101, etc.)
    - set ```virtual``` to true
    - set ```settings``` to desired outcome 

**Settings examples**
- analog port example: A13 * 23: result is whatever data is in analog 13 times 23
- analog port example: A13 + A14/2
- digital port example: (A03 > 1) && (D01==1): result is 1 if data in analog 13 > 1 and data in digital 01 is 1
- digital port example: (D01==1) && (D02==1) && (D03==0)
- digital port example: (D01==1) || (D02==1) || (A13>5.2)
- special ports
    - analog thermocouple with parameter: SP:TC(J;A13;A14)
        - SP means special port
        - TC means thermocouple
        - J is type of thermocouple  (https://www.thermocoupleinfo.com/thermocouple-types.htm)
        - A13 is thermocouple port
        - A14 is cold junction temperature port
    - analog speed with parameter: SP:SPD(D01;3;0,200)
        - SP means special port
        - SPD means speed port
        - D01 is the port, from which the speed is calculated
        - 3 means number of minutes
        - 0,200 means diameter of a circle, in meters for speed in meters per second, for continuous manufacturing
    - digital addzero port with parameter: SP:ADDZERO(D01)
        - SP means special port
        - ADDZERO means add zero port
        - D01 is the port, from which this virtual port is calculated
        - Value 1 is saved the same as the original port
        - Value 0 is saved after 0.1 seconds  


## Developer Information
Use software only as a [part of a system](https://github.com/petrjahoda/system) using Docker runtime.<br/>
 Do not run under linux, windows or mac on its own.



© 2020 Petr Jahoda



