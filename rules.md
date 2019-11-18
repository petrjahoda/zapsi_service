# Zapsi coding rules
## Naming
Software naming is lower case only with underscore.
First name is the name of the software.
Second name is the type.

**Examples**
- zapsi_service
- state_service
- terminal_service
- zapsi_webservice
- terminal_webservice
- lcd_webservice
- zapsi_postgres_database

File naming is lower case one name only.

**Examples**
- config.go
- main.go
- log.go

Variable naming is camelCase, reasonable name should be used. Use runningDevices instead of rd, runDev, ...

## Philosophy
Files should be maximal 1000 lines long, optimal <500 lines long. Comment are forbidden, use logging instead.
Always use main.go as a starting point for every software. Use default go coding conventions. Handle errors first. Do not use if-else, use only if. Use switch instead of multiples if-else.


## Git commits
Commit after every change. Use these tags:
- new
- enhancement
- change
- bug fix

Examples:
- new: table device
- enhancement: added percentage remaining for downloading data 
- change: calculating speed for special speed port (meters to centimeters)
- bug fix: logging was not working properly, not everything was logged

## Technologies

Main language : Go with libraries: GORM, httprouter, amCharts, MetroUI

Main database: PostgreSQL

Runtime: Docker

Git repository: github

Licence: MIT

## Versioning

Version contains year, quarter, month of the quarter and day of the month.

2019.2.1.24 is version from year 2019, second quarter, first month of second quarter, which is April, and from 24th of April.


## Structure
- zapsi_service downloads data from zapsi devices
- siemens_service downloadsw data from siemens plcs
- state_service generates statistics and states
- terminal_service generates automatic terminal data
- alarm_service send alarms
- zapsi_postgres_database is used as a storage
- lcd_webservice is used for displaying lcd screens
- terminal_webservice is used for displaying terminal screens
- zapsi_webservice is used for displaying main data




