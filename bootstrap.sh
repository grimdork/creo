#!/bin/sh
set -e
go build -ldflags="-s -w -X main.version=$(git describe --tags 2>/dev/null || echo dev)" -o creo
./creo -F install
