# Zapsi Service Changelog

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/).

Please note, that this project, while following numbering syntax, it DOES NOT
adhere to [Semantic Versioning](http://semver.org/spec/v2.0.0.html) rules.

## Types of changes

* ```Added``` for new features.
* ```Changed``` for changes in existing functionality.
* ```Deprecated``` for soon-to-be removed features.
* ```Removed``` for now removed features.
* ```Fixed``` for any bug fixes.
* ```Security``` in case of vulnerabilities.

## [2020.3.2.13] - 2020-08-13

### Changed
- updated to latest libraries
- updated to go 1.15 -> final program size reduced to 90%

## [2020.3.2.9] - 2020-08-09

### Fixed
- when firstly processing data... if not processed, no new are downloaded

## [2020.3.2.5] - 2020-08-05

### Added
- MIT license

### Changed
- code moved to more files
- logging updated
- name of functions updated


## [2020.3.2.4] - 2020-08-04

### Fixed
- proper NOT SAVING similar data to database, but saving different data, although in the same batch

### Changed
- update to latest libraries

## [2020.3.1.30] - 2020-07-30

### Fixed
- proper closing database connections with sqlDB, err := db.DB() and defer sqlDB.Close()

### Changed
- added tzdata to docker image

## [2020.3.1.26] - 2020-07-26

### Changed
- fastest as possible processing virtual ports
- updated processing thermocouple port
- updated replacing ports with their values

## [2020.3.1.22] - 2020-07-22

### Changed
- changed to gorm v2
- postgres only

### Removed
- all about logging to file
- config

### Security
- docker image changed from alpine to scratch

## [2020.3.1.14] - 2020-07-14

### Changed
- logging library changed to github.com/TwinProduction/go-color"

## [2020.2.2.18] - 2020-05-18

### Added
- init for actual service directory
- db.logmode(false)

## [2020.1.3.31] - 2020-03-31

### Added
- updated create.sh for uploading proper docker version automatically

## [2020.1.2.29] - 2020-02-29

### Changed
- name of database changed to zapsi3
- proper testing for mariadb, postgres and mssql
- added logging for all important methods and functions
- code refactoring for better readability

### Fixed
- properly handling parsing milliseconds from zapsi digital data