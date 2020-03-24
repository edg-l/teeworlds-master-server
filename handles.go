package main

import (
	"encoding/json"
	"net/http"
)

type Lol struct {
	Test string
}

func SetupResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Server", "TeeMaster/0.1")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

func WriteJson(w http.ResponseWriter, s interface{}) {
	js, err := json.Marshal(s)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(js)
}

func Index(w http.ResponseWriter, req *http.Request) {
	SetupResponse(w)
	WriteJson(w, &Lol{
		Test: "testing",
	})
}
