package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"
)

func main() {
	server := flag.String("server", "http://localhost:8080", "")
	size := flag.Int("size", 10*1024*1024, "download/upload size in bytes")
	flag.Parse()

	// 1. Ping
	start := time.Now()
	resp, _ := http.Get(*server + "/ping")
	latency := time.Since(start)
	// check err/status…

	// 2. Download
	start = time.Now()
	resp, _ = http.Get(fmt.Sprintf("%s/download?size=%d", *server, *size))
	n, _ := io.Copy(io.Discard, resp.Body)
	downloadTime := time.Since(start)
	speedDown := float64(n) / downloadTime.Seconds() / 1024 / 1024 // MB/s

	// 3. Upload
	payload := make([]byte, *size)
	rand.Read(payload)
	start = time.Now()
	req, _ := http.NewRequest("POST", *server+"/upload", bytes.NewReader(payload))
	resp, _ = http.DefaultClient.Do(req)
	uploadTime := time.Since(start)
	speedUp := float64(*size) / uploadTime.Seconds() / 1024 / 1024 // MB/s

	fmt.Printf("PING: %v\nDOWNLOAD: %.2f MB/s\nUPLOAD:   %.2f MB/s\n",
		latency, speedDown, speedUp)
}
