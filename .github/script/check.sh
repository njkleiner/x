#!/usr/bin/env bash

set -eu
set -o pipefail

gofmt -s -d ./ | diff -u /dev/null -
go mod tidy --diff

go vet ./...
