package main

type memcachedConfig struct {
	Host string `default:"127.0.0.1"`
	Port string `default:"11211"`
}

var config = struct {
	HeartbeatIntervalSeconds int    `default:"90"`
	Port                     uint16 `default:"8283"`
	Certificate              string `default:"./cert.pem"`
	Key                      string `default:"./key.pem"`
	Memcached                memcachedConfig
	ServerIdentifier         string `default:"Master"`
	Servers                  []string
	SocketTimeoutSeconds     int `default:"3"`
}{}
