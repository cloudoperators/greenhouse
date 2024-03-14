// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"
)

var (
	Version  = "unset"
	Revision = "unset"
)

var (
	targetURL      string
	targetCAFile   string
	host           string
	port           uint64
	origin         string
	serverLogging  bool
	version        bool
	allowedHeaders []string
	allowedMethods []string
)

func init() {
	envTargetURL := os.Getenv("CORS_REVERSE_PROXY_TARGET_URL")

	envHost := os.Getenv("CORS_REVERSE_PROXY_HOST")
	if envHost == "" {
		envHost = "0.0.0.0"
	}

	envPort := os.Getenv("CORS_REVERSE_PROXY_PORT")
	if envPort == "" {
		envPort = "8081"
	}
	portParsed, err := strconv.ParseUint(envPort, 10, 16)
	if err != nil {
		log.Fatalf("failed to parse port: %v", err)
	}

	envOrigin := os.Getenv("CORS_REVERSE_PROXY_ORIGIN")
	envServerLogging := os.Getenv("CORS_REVERSE_PROXY_SERVER_LOGGING")
	if envServerLogging == "" {
		envServerLogging = "false"
	}
	serverLoggingEnabled, err := strconv.ParseBool(envServerLogging)
	if err != nil {
		log.Fatalf("failed to parse server-logging: %v", err)
	}

	pflag.StringVar(&targetURL, "target-url", envTargetURL, "")
	pflag.StringVar(&targetCAFile, "target-ca", "", "CA file to use for connection to target-url")

	pflag.StringVar(&host, "host", envHost, "")
	pflag.Uint64Var(&port, "port", portParsed, "")
	pflag.StringVar(&origin, "origin", envOrigin, "")
	pflag.BoolVar(&serverLogging, "server-logging", serverLoggingEnabled, "")
	pflag.BoolVarP(&version, "version", "v", false, "")
	pflag.StringSliceVar(&allowedHeaders, "allowed-headers", []string{"Content-Type", "Content-Length", "Accept-Encoding", "Authorization"}, "Which headers are allowed for CORS requests")
	pflag.StringSliceVar(&allowedMethods, "allowed-methods", []string{"GET,HEAD,PUT,PATCH,POST,DELETE"}, "Which methods are allowed for CORS requests")
}

func run(targetURL string) error {
	target, err := url.Parse(targetURL)
	if err != nil {
		return err
	}

	modifyCORSResponse := func(res *http.Response) error {
		if origin := res.Request.Header.Get("Origin"); origin != "" {
			res.Header.Set("Access-Control-Allow-Methods", strings.Join(allowedMethods, ","))
			res.Header.Set("Access-Control-Allow-Credentials", "true")
			res.Header.Set("Access-Control-Allow-Headers", strings.Join(allowedHeaders, ","))
			res.Header.Set("Access-Control-Allow-Origin", origin)

			if res.Request.Method == http.MethodOptions {
				if res.Body != nil {
					//Discard the result from upstream
					_, _ = io.ReadAll(res.Body) //nolint:errcheck
				}
				res.StatusCode = 200
				res.Header.Set("Content-Length", "0")
			}
		}
		return nil
	}

	reverseProxy := httputil.NewSingleHostReverseProxy(target)
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	if targetCAFile != "" {
		cas, err := os.ReadFile(targetCAFile)
		if err != nil {
			return fmt.Errorf("failed to read target ca file%s: %w", targetCAFile, err)
		}
		caPool := x509.NewCertPool()
		if !caPool.AppendCertsFromPEM(cas) {
			return fmt.Errorf("no certs found in target CA file %s", targetCAFile)
		}
		tlsConfig.RootCAs = caPool
	}
	reverseProxy.Transport = &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	reverseProxy.ModifyResponse = modifyCORSResponse
	mux := http.NewServeMux()
	mux.Handle("/", reverseProxy)

	server := &http.Server{
		Addr:         host + ":" + strconv.FormatUint(port, 10),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      mux,
	}
	return server.ListenAndServe()
}

func printHeader() {
	log.Println("Start CORS Reverse Proxy")
	log.Println("")
	log.Printf("Target URL: %s\n", targetURL)
	log.Printf("Host: %s\n", host)
	log.Printf("Port: %d\n", port)
	log.Println("")
	log.Printf("Please access to http://%s:%d/\n", host, port)
	log.Println("")
}

func main() {
	pflag.Parse()

	if version {
		log.Printf("%s version: %s revision: %s", os.Args[0], Version, Revision)
		os.Exit(0)
	}

	if targetURL == "" {
		log.Fatal("Target URL(--target-url or -t) option is required.")
	}

	printHeader()

	if err := run(targetURL); err != nil {
		log.Fatal(err)
	}
}
