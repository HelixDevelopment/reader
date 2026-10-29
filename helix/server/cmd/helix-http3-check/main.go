// Command helix-http3-check performs a verified HTTP/3 health request.
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/quic-go/quic-go/http3"
)

func main() {
	url := flag.String("url", "", "HTTPS health URL (required)")
	caFile := flag.String("ca", "", "PEM CA certificate (required)")
	flag.Parse()
	if *url == "" || *caFile == "" {
		fmt.Fprintln(os.Stderr, "url and ca are required")
		os.Exit(2)
	}
	pem, err := os.ReadFile(*caFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(pem) {
		fmt.Fprintln(os.Stderr, "no CA certificates found")
		os.Exit(1)
	}
	transport := &http3.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS13}}
	defer transport.Close()
	client := &http.Client{Transport: transport, Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, *url, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	response, err := client.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer response.Body.Close()
	if _, err := io.ReadAll(io.LimitReader(response.Body, 1<<20)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	result := map[string]any{"protocol": response.Proto, "http_status": response.StatusCode, "status": "FAIL"}
	if response.ProtoMajor == 3 && response.StatusCode == http.StatusOK {
		result["status"] = "PASS"
	}
	_ = json.NewEncoder(os.Stdout).Encode(result)
	if result["status"] != "PASS" {
		os.Exit(1)
	}
}
