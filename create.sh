#!/usr/bin/env bash
cd linux
upx zapsi_service_linux
cd ..
cd mac
upx zapsi_service_mac
cd ..
cd windows
upx zapsi_service_windows.exe
cd ..
docker rmi -f petrjahoda/zapsi_service:"$1"
docker build -t petrjahoda/zapsi_service:"$1" .
docker push petrjahoda/zapsi_service:"$1"