#!/bin/sh

# ------------------------------------
# Purpose:
# - Build notice file for all FOSS modules.
#
# Releases:
# - v1.0.0 - 2026-04-21: initial release
#
# Remarks:
# - Requirements: go-licenses, notice template
# ------------------------------------

# set -o xtrace
set -o verbose
set -o errexit

# update vendoring directory
go mod vendor 

# build notice file based on template
go-licenses report . --template notice.tpl > NOTICE
