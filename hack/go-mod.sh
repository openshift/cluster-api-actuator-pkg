#!/bin/bash

pushd $1 > /dev/null
go mod tidy
go mod vendor
go mod verify
popd > /dev/null
