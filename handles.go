package main

import (
	"encoding/json"
	"net"
	"net/http"
	"time"
)

type ServerPayload struct {
	Port uint16 `json:"port"`
}

var serverStore = &ServerStore{
	Servers: make(map[ServerKey]*ServerEntry),
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

func ClearStore() {
	serverStore.Lock()
	defer serverStore.Unlock()
	serverStore.Servers = make(map[ServerKey]*ServerEntry)
}

func Index(w http.ResponseWriter, req *http.Request) {
	setupResponse(w)

	if req.Method == http.MethodPost {
		if req.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}

		server := ServerPayload{}

		decoder := json.NewDecoder(req.Body)
		decoder.DisallowUnknownFields()

		err := decoder.Decode(&server)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		serverStore.Lock()
		defer serverStore.Unlock()

		host, _, err := net.SplitHostPort(req.RemoteAddr)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		key := ServerKey{
			Address: host,
			Port:    server.Port,
		}

		if _, ok := serverStore.Servers[key]; ok {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		expire := time.Now().Add(time.Second * time.Duration(Config.HeartbeatIntervalSeconds))

		serverStore.Servers[key] = &ServerEntry{
			Expire: expire,
		}
		// TODO: Enforce limit

		w.WriteHeader(http.StatusCreated)
		writeJSON(w, &map[string]interface{}{
			"expire": expire.Unix(),
		})

		return
	}
	w.WriteHeader(http.StatusNotFound)
}

func Heartbeat(w http.ResponseWriter, req *http.Request) {
	setupResponse(w)

	if req.Method == http.MethodPost {
		if req.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}

		server := ServerPayload{}

		decoder := json.NewDecoder(req.Body)
		decoder.DisallowUnknownFields()

		err := decoder.Decode(&server)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		serverStore.Lock()
		defer serverStore.Unlock()

		host, _, err := net.SplitHostPort(req.RemoteAddr)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		key := ServerKey{
			Address: host,
			Port:    server.Port,
		}

		if entry, ok := serverStore.Servers[key]; ok {
			// Don't allow a heartbeat if it's too early to prevent spam.
			if time.Now().Before(entry.Expire.Add(-time.Second * time.Duration(Config.HeartbeatIntervalSeconds-Config.HeartbeatMinWaitSeconds))) {
				w.WriteHeader(http.StatusForbidden)
				return
			}

			entry.Expire = time.Now().Add(time.Second * time.Duration(Config.HeartbeatIntervalSeconds))
			w.WriteHeader(http.StatusOK)
			return
		}

		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNotFound)
}
