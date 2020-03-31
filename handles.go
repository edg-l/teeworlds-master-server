package main

import (
	"encoding/json"
	"net"
	"net/http"
	"time"
)

type serverPayload struct {
	Port  uint16 `json:"port"`
	Token string `json:"token"`
}

func setupResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Server", "TeeMaster/0.1")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	w.Header().Set("X-Content-Type-Options", "nosniff")
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

		if err != nil || server.Port == 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		host, _, err := net.SplitHostPort(req.RemoteAddr)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		ip := net.ParseIP(host)

		if ip == nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		exits, err := addressExists(ip)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if exits {
			w.WriteHeader(http.StatusConflict)
			return
		}

		store.Lock()
		defer store.Unlock()

		if server.Token != "" {
			key := serverKey{Token: server.Token}
			entry, found := store.Servers[key]

			if !found {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			// You shouldn't use a token when registering a server (not adding a ipv6 or ipv4 address).
			if entry.Port != server.Port {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			if ip.To4() != nil {
				entry.Address4 = ip
			} else {
				entry.Address6 = ip
			}
			w.WriteHeader(http.StatusCreated)
			go saveListToCache()
			return
		}

		key := makeKey()

		// Make sure the key is unique
		for {
			if _, ok := store.Servers[key]; !ok {
				break
			}
			key = makeKey()
		}

		expire := time.Now().Add(time.Second * time.Duration(config.HeartbeatIntervalSeconds))

		entry := &serverEntry{
			Expire: expire,
			Port:   server.Port,
		}

		if ip.To4() != nil {
			entry.Address4 = ip
		} else {
			entry.Address6 = ip
		}

		store.Servers[key] = entry

		// Create a goroutine here, so when the defered mutex close gets called, this will proceed.
		go saveListToCache()

		w.WriteHeader(http.StatusCreated)
		writeJSON(w, &map[string]interface{}{
			"token": key.Token,
		})

		return
	} else if req.Method == http.MethodGet {

		entries, _ := getServerList()

		b, err := json.Marshal(entries)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Write(b)
		return
	}
	w.WriteHeader(http.StatusNotFound)
}
