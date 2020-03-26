package main

import (
	"sync"
	"time"
)

type serverKey struct {
	Address string `json:"address"`
	Port    uint16 `json:"port"`
}

type serverEntry struct {
	Expire time.Time
}

type serverStore struct {
	sync.RWMutex
	Servers map[serverKey]*serverEntry
}
