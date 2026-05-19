// Command healthcheck performs an HTTP health probe and exits 0 on success.
// Used as HEALTHCHECK in the distroless Docker image (no shell/busybox available).
package main

import (
	"net/http"
	"os"
)

func main() {
	url := "http://localhost:8080/health/ready"
	if len(os.Args) > 1 {
		url = os.Args[1]
	}
	resp, err := http.Get(url) //nolint:gosec,noctx
	if err != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
		os.Exit(1)
	}
}
