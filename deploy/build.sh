#!/bin/bash

go test -v ../... || exit 1

GOOS=linux GOARCH=amd64 go build -v -o ./sfsol ../cmd/sfsol


TAG=gcr.io/eoscanada-shared-services/sfsol
REV=$(git rev-parse --short HEAD)
IMG=$TAG:ab-$REV

docker build . -t $IMG

echo
echo Get ready
echo
echo kcns eth-dev1
echo kubectl set image sts sfsol=$IMG
echo

docker push $IMG

echo
echo kcns eth-dev1
echo kubectl set image sts sfsol=$IMG
echo
