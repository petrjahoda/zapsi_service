#!/usr/bin/env bash
docker rmi -f petrjahoda/zapsi_service:"$1"
docker build -t petrjahoda/zapsi_service:"$1" .
docker push petrjahoda/zapsi_service:"$1"