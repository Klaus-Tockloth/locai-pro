#!/bin/sh

# ------------------------------------
# Purpose:
# - Builds uploads (tar.gz or zip) for Github project repository (assets in release section).
#
# Releases:
# - v1.0.0 - 2026-04-21: initial release
# ------------------------------------

# set -o xtrace
set -o verbose

# recreate directory
rm -r ./uploads
mkdir ./uploads

# uploads 'darwin'
# tar -cvzf ./uploads/macos-amd64_locai-pro.tar.gz ./binaries/darwin-amd64/locai-pro
tar -cvzf ./uploads/macos-arm64_locai-pro.tar.gz ./binaries/darwin-arm64/locai-pro

# uploads 'linux'
tar -cvzf ./uploads/linux-amd64_locai-pro.tar.gz ./binaries/linux-amd64/locai-pro
tar -cvzf ./uploads/linux-arm64_locai-pro.tar.gz ./binaries/linux-arm64/locai-pro

# uploads 'windows'
zip ./uploads/windows-amd64_locai-pro.zip ./binaries/windows-amd64/locai-pro.exe
zip ./uploads/windows-arm64_locai-pro.zip ./binaries/windows-arm64/locai-pro.exe

# uploads 'freebsd'
# tar -cvzf ./uploads/freebsd-amd64_locai-pro.tar.gz ./binaries/freebsd-amd64/locai-pro
# tar -cvzf ./uploads/freebsd-arm64_locai-pro.tar.gz ./binaries/freebsd-arm64/locai-pro

# uploads 'netbsd'
# tar -cvzf ./uploads/netbsd-amd64_locai-pro.tar.gz ./binaries/netbsd-amd64/locai-pro
# tar -cvzf ./uploads/netbsd-arm64_locai-pro.tar.gz ./binaries/netbsd-arm64/locai-pro

# uploads 'openbsd'
# tar -cvzf ./uploads/openbsd-amd64_locai-pro.tar.gz ./binaries/openbsd-amd64/locai-pro
# tar -cvzf ./uploads/openbsd-arm64_locai-pro.tar.gz ./binaries/openbsd-arm64/locai-pro

