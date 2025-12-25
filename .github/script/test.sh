#!/usr/bin/env bash

set -eu
set -o pipefail

go clean -testcache

go test -cover -race -vet=off -v ./...
