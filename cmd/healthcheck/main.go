package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

// healthcheck is invoked by the container HEALTHCHECK. Keeping it as a tiny Go
// binary keeps curl (and therefore apt) out of the runtime image, so production
// image builds do not depend on Debian package mirrors.
func main() {
	url := "http://127.0.0.1:8080/health"
	if len(os.Args) > 1 {
		url = os.Args[1]
	}
	client := &http.Client{Timeout: 3 * time.Second}
	response, err := client.Get(url)
	if err != nil {
		fmt.Fprintln(os.Stderr, "healthcheck:", err)
		os.Exit(1)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		fmt.Fprintln(os.Stderr, "healthcheck: unexpected status", response.StatusCode)
		os.Exit(1)
	}
}
