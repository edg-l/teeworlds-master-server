package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/jinzhu/configor"
	"github.com/urfave/cli/v2"
)

/*
	TODO:
	- https://github.com/teeworlds/teeworlds/issues/1703
	- https://github.com/teeworlds/teeworlds/issues/713
	- https://github.com/teeworlds/teeworlds/issues/1292
*/

var memcacheClient *memcache.Client

func main() {
	memcache.New()
	var certPath string
	var keyPath string
	var port int

	configor.Load(&config, "config.yml")

	app := &cli.App{
		Name:  "HTTPS Teeworlds Master Server",
		Usage: "A implementation of the HTTPS teeworlds master server in Go.",
		Commands: []*cli.Command{
			&cli.Command{
				Name:    "generate",
				Aliases: []string{"g", "gen"},
				Usage:   "Generates a certificate.",
				Action: func(c *cli.Context) error {
					fmt.Println("Generating certificate...")
					priv, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)

					if err != nil {
						log.Panicf("Failed to generate ECDSA key: %s\n", err)
					}

					template := x509.Certificate{
						SerialNumber: big.NewInt(1),
						Subject: pkix.Name{
							CommonName: "Teeworlds",
						},
						NotBefore: time.Now(),
						NotAfter:  time.Now().Add(time.Hour * 24 * 365),

						KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
						ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
						BasicConstraintsValid: true,
					}

					derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
					if err != nil {
						log.Panicf("Failed to create certificate: %s\n", err)
					}

					out := &bytes.Buffer{}
					pem.Encode(out, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
					err = ioutil.WriteFile("cert.pem", out.Bytes(), 0600)
					println("Created cert.pem")
					out.Reset()

					b, err := x509.MarshalECPrivateKey(priv)
					if err != nil {
						log.Panicf("Unable to marshal ECDSA private key: %s\n", err)
					}
					block := &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}
					pem.Encode(out, block)

					err = ioutil.WriteFile("key.pem", out.Bytes(), 0600)
					println("Created key.pem")

					if err != nil {
						log.Panicf("Failed to save certificate: %s\n", err)
					}
					return nil
				},
			},
			&cli.Command{
				Name:    "start",
				Aliases: []string{"s"},
				Flags: []cli.Flag{
					&cli.PathFlag{
						Name:        "key",
						Aliases:     []string{"k"},
						Value:       config.Key,
						Destination: &keyPath,
					},
					&cli.PathFlag{
						Name:        "cert",
						Aliases:     []string{"c"},
						Value:       config.Certificate,
						Destination: &certPath,
					},
					&cli.IntFlag{
						Name:        "port",
						Aliases:     []string{"p"},
						Value:       int(config.Port),
						Destination: &port,
					},
				},
				Action: func(c *cli.Context) error {
					absCertPath, err := filepath.Abs(certPath)

					if err != nil {
						panic(err)
					}

					absKeyPath, err := filepath.Abs(keyPath)

					if err != nil {
						panic(err)
					}

					mux := http.NewServeMux()

					mux.HandleFunc("/", index)

					log.Print("Connecting to memcached...\n")

					memcacheClient = memcache.New(fmt.Sprintf("%s:%s", config.Memcached.Host, config.Memcached.Port))
					if memcacheClient.Ping() != nil {
						// TODO: Maybe make a mode where you can run without memcached.
						log.Fatalf("Failed to connect to memcached")
					}

					log.Print("Connected to memcached.")

					saveListToCache()

					log.Println("Starting HTTPS Master server on port", port)

					err = http.ListenAndServeTLS(fmt.Sprintf(":%d", port), absCertPath, absKeyPath, mux)

					if err != nil {
						log.Fatal("ListenAndServeTLS: ", err)
					}

					return nil
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
