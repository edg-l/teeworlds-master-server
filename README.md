# HTTPS Teeworlds Master Server ![Go](https://github.com/Ryozuki/teeworlds-master-server/workflows/Go/badge.svg)

The masterserver uses memcached to cache the server entries it receives, thus multiple master servers can be used.

Each master server has a configuration `config.yml` where you define a identifier, and also define which other master server entries to use by listing their identifiers.

Each master server uses it's identifier as cache key where it saves his managed list of servers.

When a master server must provide the server list, it will then use the server identifiers as the cache keys to get the full server list.

Each master server manages his own registered server list, in the cache only relevant info for the client should be saved and not for example, when the server will timeout in the list.

## Build

`go build`

## Generate certificate

Generate a self-signed key:

`./teeworlds-master-server generate`

*Note: this cert will only last 1 year.*

## Dependencies

You need to install https://memcached.org/

You should limit the connection to the memcached server on a firewall level.

## Start

`./teeworlds-master-server start`

## TODO

Colorize a bit? https://github.com/logrusorgru/aurora

## Util

`curl -k https://localhost:8283/`

`http --verify=no https://localhost:8283/`
