#!/usr/bin/env bash
docker rmi -f petrjahoda/zapsi-service:"$1"
docker build -t petrjahoda/zapsi-service:"$1" .
docker push petrjahoda/zapsi-service:"$1"