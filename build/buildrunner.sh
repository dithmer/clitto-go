#!/bin/bash
set -xeuo pipefail
go get -d -v
NAME="$1"
export NAME
export GOOS
export GOARCH
while read -r target
do
	GOOS="${target%% *}"
	GOARCH="${target#* }"
        go build -v -o "dist-$NAME-$GOOS-$GOARCH"
done < "$(dirname "$0")/targets"
