package main

import (
	"crypto/rand"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/labstack/gommon/log"
	"github.com/vmihailenco/msgpack"
)

type serverKey struct {
	Token string
}

func makeKey() serverKey {
	b := make([]byte, 16)
	rand.Read(b)
	return serverKey{
		Token: fmt.Sprintf("%x", b),
	}
}

type serverEntry struct {
	Address4 net.IP    `json:"address4"`
	Address6 net.IP    `json:"address6"`
	Port     uint16    `json:"port"`
	Expire   time.Time `json:"-"`
}

type serverStore struct {
	sync.RWMutex
	Servers map[serverKey]*serverEntry
}

var store = &serverStore{
	Servers: make(map[serverKey]*serverEntry),
}

func addressExists(address net.IP) (bool, error) {
	entries, err := getServerList()

	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		if entry.Address4.Equal(address) || entry.Address6.Equal(address) {
			return true, nil
		}
	}

	return false, nil
}

func getServerEntries() []*serverEntry {
	store.RLock()
	defer store.RUnlock()

	entries := make([]*serverEntry, 0, len(store.Servers))
	for _, entry := range store.Servers {
		entries = append(entries, entry)
	}

	return entries
}

func getEntriesFromCache(ident string) ([]serverEntry, error) {
	item, err := memcacheClient.Get(ident)

	if err != nil {
		return []serverEntry{}, err
	}

	var entries []serverEntry

	err = msgpack.Unmarshal(item.Value, &entries)

	return entries, err
}

func saveListToCache() bool {
	entries := getServerEntries()

	b, err := msgpack.Marshal(entries)
	if err != nil {
		log.Warn("Could not marshal cache list: ", err)
		return false
	}

	memcacheClient.Set(&memcache.Item{
		Key:   config.ServerIdentifier,
		Value: b,
	})

	return true
}

func getServerList() ([]serverEntry, error) {
	totalEntries, err := getEntriesFromCache(config.ServerIdentifier)

	if err != nil {
		return []serverEntry{}, err
	}

	for _, ident := range config.Servers {
		entries, err := getEntriesFromCache(ident)

		// If we can't get the cached list from another master server, continue gracefully.
		if err != nil {
			continue
		}

		totalEntries = append(totalEntries, entries...)
	}

	return totalEntries, nil
}
