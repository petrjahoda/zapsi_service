![Software](https://img.shields.io/badge/CLI-Zapsi%20Service-blue) 
![Go version](https://img.shields.io/badge/GO-1.15-brightgreen)
![License](https://img.shields.io/badge/License-MIT-brightgreen)


# Zapsi Service


## Installation
* use docker image from https://cloud.docker.com/r/petrjahoda/zapsi_service
* use linux, mac or windows version and make it run like a service (on windows use nssm)

## Description
Go service that download data from Zapsi devices and insert them to database.

## Additional information
* realtime evaluation of virtual ports:
    * analog port example: A13 * 23: result is whatever is A13 times 23
    * analog port example: A13 + A14/2
    * digital port example: (A03 > 1) && (D01==1): result is 1 if A13>1 and D01 is 1
    * digital port example: (D01==1) && (D02==1) && (D03==0)
    * digital port example: (D01==1) || (D02==1) || (A13>5.2)
* special ports
    * analog thermocouple with parameter: SP:TC(J;A13;A14)
    * analog speed with parameter: SP:SPD(D01;3;0,200)
    * digital addzero port with parameter: SP:ADDZERO(D01)

### Special Ports  
* Thermocouple    
    * SP means special port
    * TC means thermocouple
    * J is type of thermocouple  (https://www.thermocoupleinfo.com/thermocouple-types.htm)
    * A13 is thermocouple port
    * A14 is cold junction temperature port
* Speed     
    * SP means special port
    * SPD means speed port
    * D01 is the port, from which the speed is calculated
    * 3 means number of minutes
    * 0,200 means diameter of a circle, in meters for speed in meters per second, for continuous manufacturing
* Zero Port     
    * SP means special port
    * ADDZERO means add zero port
    * D01 is the port, from which this virtual port is calculated
    * Value 1 is saved the same as the original port
    * Value 0 is saved after 0.1 seconds

    
tGMS © 2020 Petr Jahoda
