package main

import (
	"encoding/json"
	"net"
	"net/http"
	"time"
)

type ServerRegisterPayload struct {
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

		server := ServerRegisterPayload{}

		decoder := json.NewDecoder(req.Body)
		decoder.DisallowUnknownFields()

		err := decoder.Decode(&server)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		serverStore.RLock()

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
			serverStore.RUnlock()
			w.WriteHeader(http.StatusForbidden)
			return
		}

		serverStore.RUnlock()

		serverStore.Lock()
		defer serverStore.Unlock()

		expire := time.Now().Add(time.Second * 90)

		serverStore.Servers[key] = &ServerEntry{
			Expire: expire,
		}

		w.WriteHeader(http.StatusCreated)
		writeJSON(w, &map[string]interface{}{
			"expire": expire.Unix(),
		})

		return
	}
	w.WriteHeader(http.StatusNotFound)
}
