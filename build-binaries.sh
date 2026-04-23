#!/bin/sh

# ------------------------------------
# Purpose:
# - Build binaries for supported target systems.
#
# Releases:
# - v1.0.0 - 2026-04-21: initial release
# ------------------------------------

set -o errexit
set -v -o verbose

# recreate directory
rm -r ./binaries
mkdir ./binaries

# renew vendor content
go mod tidy
go mod vendor

# lint
golangci-lint run --no-config --enable gocritic
revive

# security
govulncheck ./...
gosec -exclude=G114,G115,G204,G302,G304 ./...

# show compiler version
go version

# compile 'darwin' (macOS)
# env GOOS=darwin GOARCH=amd64 go build -v -o binaries/darwin-amd64/locai-pro
env GOOS=darwin GOARCH=arm64 go build -v -o binaries/darwin-arm64/locai-pro

# compile 'linux'
env GOOS=linux GOARCH=amd64 go build -v -o binaries/linux-amd64/locai-pro
env GOOS=linux GOARCH=arm64 go build -v -o binaries/linux-arm64/locai-pro

# compile 'windows'
env GOOS=windows GOARCH=amd64 go build -v -o binaries/windows-amd64/locai-pro.exe
env GOOS=windows GOARCH=arm64 go build -v -o binaries/windows-arm64/locai-pro.exe

# compile 'freebsd'
# env GOOS=freebsd GOARCH=amd64 go build -v -o binaries/freebsd-amd64/locai-pro
# env GOOS=freebsd GOARCH=arm64 go build -v -o binaries/freebsd-arm64/locai-pro

# compile 'openbsd'
# env GOOS=openbsd GOARCH=amd64 go build -v -o binaries/openbsd-amd64/locai-pro
# env GOOS=openbsd GOARCH=arm64 go build -v -o binaries/openbsd-arm64/locai-pro

# compile 'netbsd'
# env GOOS=netbsd GOARCH=amd64 go build -v -o binaries/netbsd-amd64/locai-pro
# env GOOS=netbsd GOARCH=arm64 go build -v -o binaries/netbsd-arm64/locai-pro

