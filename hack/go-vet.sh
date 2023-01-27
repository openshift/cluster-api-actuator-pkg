#!/bin/sh

pushd $1 > /dev/null
go vet "./..."
popd > /dev/null
