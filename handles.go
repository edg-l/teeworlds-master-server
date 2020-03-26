package main

import (
	"encoding/json"
	"net"
	"net/http"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/labstack/gommon/log"
	"github.com/vmihailenco/msgpack/v4"
)

type serverPayload struct {
	Port uint16 `json:"port"`
}

var store = &serverStore{
	Servers: make(map[serverKey]*serverEntry),
}

func setupResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Server", "TeeMaster/0.1")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

func getServerKeys() []serverKey {
	store.RLock()
	defer store.RUnlock()

	keys := make([]serverKey, 0, len(store.Servers))
	for key := range store.Servers {
		keys = append(keys, key)
	}

	return keys
}

func getKeysFromCache(ident string) ([]serverKey, error) {
	item, err := memcacheClient.Get(ident)

	if err != nil {
		return []serverKey{}, err
	}

	var keys []serverKey

	err = msgpack.Unmarshal(item.Value, &keys)

	return keys, err
}

func saveListToCache() bool {
	keys := getServerKeys()

	b, err := msgpack.Marshal(keys)
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

func getServerList() ([]byte, error) {
	ourKeys, err := getKeysFromCache(config.ServerIdentifier)

	if err != nil {
		return []byte{}, err
	}

	for _, ident := range config.Servers {
		keys, err := getKeysFromCache(ident)

		// If we can't get the cached list from another master server, continue gracefully.
		if err != nil {
			continue
		}

		ourKeys = append(ourKeys, keys...)
	}

	js, err := json.Marshal(ourKeys)
	return js, err
}

func writeJSON(w http.ResponseWriter, s interface{}) bool {
	js, err := json.Marshal(s)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return false
	}

	w.Write(js)
	return true
}

func clearStore() {
	store.Lock()
	defer store.Unlock()
	store.Servers = make(map[serverKey]*serverEntry)
}

func index(w http.ResponseWriter, req *http.Request) {
	setupResponse(w)

	if req.Method == http.MethodPost {
		if req.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}

		server := serverPayload{}

		decoder := json.NewDecoder(req.Body)
		decoder.DisallowUnknownFields()

		err := decoder.Decode(&server)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		store.Lock()
		defer store.Unlock()

		host, _, err := net.SplitHostPort(req.RemoteAddr)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		key := serverKey{
			Address: host,
			Port:    server.Port,
		}

		if _, ok := store.Servers[key]; ok {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		expire := time.Now().Add(time.Second * time.Duration(config.HeartbeatIntervalSeconds))

		store.Servers[key] = &serverEntry{
			Expire: expire,
		}

		// Create a goroutine here, so when the defered mutex close gets called, this will proceed.
		go saveListToCache()

		// TODO: Enforce limit

		w.WriteHeader(http.StatusCreated)
		writeJSON(w, &map[string]interface{}{
			"expire": expire.Unix(),
		})

		return
	} else if req.Method == http.MethodGet {

		keys, _ := getServerList()

		w.Write(keys)
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

func heartbeat(w http.ResponseWriter, req *http.Request) {
	setupResponse(w)

	if req.Method == http.MethodPost {
		if req.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}

		server := serverPayload{}

		decoder := json.NewDecoder(req.Body)
		decoder.DisallowUnknownFields()

		err := decoder.Decode(&server)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		store.Lock()
		defer store.Unlock()

		host, _, err := net.SplitHostPort(req.RemoteAddr)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		key := serverKey{
			Address: host,
			Port:    server.Port,
		}

		if entry, ok := store.Servers[key]; ok {
			// Don't allow a heartbeat if it's too early to prevent spam.
			if time.Now().Before(entry.Expire.Add(-time.Second * time.Duration(config.HeartbeatIntervalSeconds-config.HeartbeatMinWaitSeconds))) {
				w.WriteHeader(http.StatusForbidden)
				return
			}

			entry.Expire = time.Now().Add(time.Second * time.Duration(config.HeartbeatIntervalSeconds))
			w.WriteHeader(http.StatusOK)
			return
		}

		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNotFound)
}
