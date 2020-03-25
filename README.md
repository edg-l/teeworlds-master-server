# HTTPS Teeworlds Master Server ![Go](https://github.com/Ryozuki/teeworlds-master-server/workflows/Go/badge.svg)

## Build

`go build`

## Generate certificate

Generate a self-signed key:

`./teeworlds-master-server generate`

*Note: this cert will only last 1 year.*

## Start

`./teeworlds-master-server start`

## TODO

Colorize a bit? https://github.com/logrusorgru/aurora

## Util

`curl -k https://localhost:8283/`

`http --verify=no https://localhost:8283/`
