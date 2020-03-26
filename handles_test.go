package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type createResponse struct {
	Expire int `json:"expire"`
}

func TestMaster(t *testing.T) {
	t.Cleanup(clearStore)
	ts := httptest.NewServer(http.HandlerFunc(index))
	defer ts.Close()

	body, _ := json.Marshal(&map[string]interface{}{
		"Port": 8303,
	})

	res, err := http.Post(ts.URL, "application/json", bytes.NewReader(body))

	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != http.StatusCreated {
		t.Fatal("Invalid status code: ", res.StatusCode)
	}

	decoder := json.NewDecoder(res.Body)
	decoder.DisallowUnknownFields()

	createRes := createResponse{}

	err = decoder.Decode(&createRes)

	if err != nil {
		t.Fatal("Error decoding response: ", err)
	}

	res, err = http.Post(ts.URL, "application/json", bytes.NewReader([]byte("}invalid json}{}{{")))

	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != http.StatusBadRequest {
		t.Fatal("Handler didn't detect invalid JSON.")
	}

	res, err = http.Post(ts.URL, "text/plain", bytes.NewReader([]byte("hello world!")))

	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatal("Handler didn't detect invalid mime type.")
	}

}

func BenchmarkAllPortRange(b *testing.B) {
	b.Cleanup(clearStore)
	ts := httptest.NewServer(http.HandlerFunc(index))
	defer ts.Close()
	b.ResetTimer()

	for port := uint16(1); int(port) < b.N && port < ^uint16(0); port++ {
		body, _ := json.Marshal(&map[string]interface{}{
			"Port": port,
		})

		res, err := http.Post(ts.URL, "application/json", bytes.NewReader(body))

		if err != nil {
			b.Fatal(err)
		}

		if res.StatusCode != http.StatusCreated {
			b.Fatal("Invalid status code: ", res.Status)
		}

		decoder := json.NewDecoder(res.Body)
		decoder.DisallowUnknownFields()

		createRes := createResponse{}

		err = decoder.Decode(&createRes)

		if err != nil {
			b.Fatal("Error decoding response: ", err)
		}
	}
}
