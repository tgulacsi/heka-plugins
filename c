#!/bin/sh
set -e
GOPATH=$(cd $(dirname $0) && pwd)/.go:$GOPATH
echo fmt
go fmt
echo vet
go vet
echo lint
if which golint 2>/dev/null; then
    golint *.go
fi
echo build
go build
#-tags nolua
echo test
go test -tags nolua "$@"
