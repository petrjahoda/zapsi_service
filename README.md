# Zapsi Service


## Installation
* use docker image from https://cloud.docker.com/repository/docker/petrjahoda/zapsi_service
* use linux, mac or windows version and make it run like a service

## Description
Go service that download data from Zapsi devices and insert them to database.

## Additional information
* ready to run in docker (linux, mac and windows service also available)
* no need to restart when new device is added/removed
* no need to restart when new device_port is added/removed
* checking database availability w/ email when unavailable
* using intermediate file for faster communication with device
* using JSON config file for even better configurability
* ability to select, which device_types to download from
* usable for all zapsi devices (two formats of datetime)
* usable for all length of data in digital.txt, analog.txt, serial.txt and ui_value.txt
* realtime evaluation of virtual ports:
    * analog port example: A103 * 23: result is whatever is A103 times 23
    * analog port example: A103 + A104/2
    * digital port example: (A103 > 1) && (D1==1): result is 1 if A103>1 and D1 is 1
    * digital port example: (D1==1) && (D2==1) && (D3==0)
    * digital port example: (D1==1) || (D2==1) || (A103>5.2)
* special ports
    * analog thermocouple with parameter: SP:TC(J;A103;A104)
    * analog speed with parameter: SP:SPD(D1;3;0,200)
    * digital addzero port with parameter: SP:ADDZERO(D1)

### Special Ports  
* Thermocouple    
    * SP means special port
    * TC means thermocouple
    * J is type of thermocouple  (https://www.thermocoupleinfo.com/thermocouple-types.htm)
    * A103 is thermocouple port
    * A104 is cold junction temperature port
* Speed     
    * SP means special port
    * SPD means speed port
    * D1 is the port, from which the speed is calculated
    * 3 means number of minutes
    * 0,200 means diameter of a circle, in meters for speed in meters per second, for continuous manufacturing
* Zero Port     
    * SP means special port
    * ADDZERO means add zero port
    * D1 is the port, from which this virtual port is calculated
    * Value 1 is saved the same as the original port
    * Value 0 is saved after 0.1 seconds

    
www.zapsi.eu © 2020
