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

var fwCheckPacket = []byte{255, 255, 255, 255, 'f', 'w', '?', '?'}
var fwCheckResponsePacket = []byte{255, 255, 255, 255, 'f', 'w', '!', '!'}
var fwOkPacket = []byte{255, 255, 255, 255, 'f', 'w', 'o', 'k'}
var fwErrPacket = []byte{255, 255, 255, 255, 'f', 'w', 'e', 'r'}

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

func packetEquals(a []byte, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func checkServer(address string, port uint16, ipv6 bool) bool {
	format := "%s:%d"

	if ipv6 {
		format = "[%s]:%d"
	}

	raddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(format, address, port))

	if err != nil {
		log.Errorf("Error resolving udp address %s:%u -> %s\n", address, port, err.Error())
		return false
	}

	conn, err := net.DialUDP("udp", nil, raddr)

	if err != nil {
		log.Errorf("Error connecting to udp address %s:%u -> %s\n", address, port, err.Error())
		return false
	}

	defer conn.Close()

	conn.SetWriteDeadline(time.Now().Add(time.Second * time.Duration(config.SocketTimeoutSeconds)))
	_, err = fmt.Fprint(conn, fwCheckPacket)

	buffer := make([]byte, len(fwCheckResponsePacket))

	conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(config.SocketTimeoutSeconds)))
	_, _, err = conn.ReadFrom(buffer)

	conn.SetWriteDeadline(time.Now().Add(time.Second * time.Duration(config.SocketTimeoutSeconds)))
	if err != nil {
		log.Errorf("Timeout/Error on %s:%u -> %s\n", address, port, err.Error())
		fmt.Fprint(conn, fwErrPacket)
		return false
	}

	if packetEquals(buffer, fwCheckResponsePacket) {
		log.Info("Received fwcheck response.")
		fmt.Fprint(conn, fwOkPacket)
		return true
	}

	log.Info("Received invalid fwcheck response: ", buffer)
	fmt.Fprint(conn, fwErrPacket)
	return false
}
