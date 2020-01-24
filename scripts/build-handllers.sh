#!/bin/bash

dir=$(cd $(dirname $0); pwd)

find cmd -name main.go -type f \
 | xargs -n 1 dirname \
 | xargs -n 1 -I@ bash -c "cd ./@ && CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=6 go build -v -installsuffix cgo -o ${dir}/../build/deploy/@/main . && pwd"

