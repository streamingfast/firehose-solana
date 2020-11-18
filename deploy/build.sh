#!/bin/bash

go test -v ../... || exit 1

GOOS=linux GOARCH=amd64 go build -v -o ./dfusesol ../cmd/dfusesol


TAG=gcr.io/eoscanada-shared-services/dfusesol
REV=$(git rev-parse --short HEAD)
IMG=$TAG:ab-$REV

docker build . -t $IMG

echo
echo Get ready
echo
echo kcns eth-dev1
echo kubectl set image sts dfusesol=$IMG
echo

docker push $IMG

echo
echo kcns eth-dev1
echo kubectl set image sts dfusesol=$IMG
echo
