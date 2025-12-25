#!/usr/bin/env bash

set -eu
set -o pipefail

go mod download
go mod verify
