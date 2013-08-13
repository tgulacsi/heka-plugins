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
find . -maxdepth 1 -type d -name '[a-z]*' | while read dn; do
    echo go build $dn
    go build $dn
done
#-tags nolua
echo test
go test -tags nolua "$@"
