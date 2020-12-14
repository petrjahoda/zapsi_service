#!/usr/bin/env bash
./update
name=${PWD##*/}
go get -u all
GOOS=linux go build -ldflags="-s -w" -o linux/"$name"
cd linux
upx "$name"
cd ..

docker rmi -f petrjahoda/"$name":latest
docker  build -t petrjahoda/"$name":latest .
docker push petrjahoda/"$name":latest

docker rmi -f petrjahoda/"$name":2020.4.3
docker build -t petrjahoda/"$name":2020.4.3 .
docker push petrjahoda/"$name":2020.4.3
