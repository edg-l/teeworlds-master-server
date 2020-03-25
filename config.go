package main

var Config = struct {
	HeartbeatIntervalSeconds int    `default:"90"`
	HeartbeatMinWaitSeconds  int    `default:"70"`
	Port                     uint16 `default:"8283"`
	Certificate              string `default:"./cert.pem"`
	Key                      string `default:"./key.pem"`
}{}
