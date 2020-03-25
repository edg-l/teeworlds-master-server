package main

import (
	"sync"
	"time"
)

type ServerKey struct {
	Address string
	Port    uint16
}

type ServerEntry struct {
	Expire time.Time
}

type ServerStore struct {
	sync.RWMutex
	Servers map[ServerKey]*ServerEntry
}
