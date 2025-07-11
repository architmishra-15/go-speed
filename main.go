package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

var (
	payloadChunk []byte
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// request details and duration
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("Started %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
		dur := time.Since(start)
		log.Printf("Completed %s %s in %v", r.Method, r.URL.Path, dur)
	})
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("pong"))
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	sizeParam := r.URL.Query().Get("size")
	var totalSize int
	var err error
	if sizeParam == "" {
		totalSize = 40 * 1024 * 1024 // 40 MB
	} else {
		totalSize, err = strconv.Atoi(sizeParam)
		if err != nil || totalSize <= 0 {
			log.Printf("Invalid size parameter: %q", sizeParam)
			http.Error(w, "invalid size parameter", http.StatusBadRequest)
			return
		}
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.Itoa(totalSize))

	// Stream by repeating the pre-generated chunk
	var sent int
	chunkSize := len(payloadChunk)
	for sent < totalSize {
		n := totalSize - sent
		if n > chunkSize {
			n = chunkSize
		}
		if _, err := w.Write(payloadChunk[:n]); err != nil {
			log.Printf("Error writing chunk: %v", err)
			return
		}
		sent += n
	}
	log.Printf("Served /download size=%d bytes", totalSize)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	bytesRead, err := io.Copy(io.Discard, r.Body)
	if err != nil {
		log.Printf("Error reading upload body: %v", err)
		http.Error(w, "error reading body", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("received %d bytes", bytesRead)))
	log.Printf("Handled /upload: %d bytes received", bytesRead)
}

func main() {
	port := flag.Int("port", 8080, "port to run the speedtest server on")
	rTimeout := flag.Duration("read-timeout", 10*time.Second, "HTTP server read timeout")
	wTimeout := flag.Duration("write-timeout", 0, "HTTP server write timeout (0 for no limit)")
	idleTimeout := flag.Duration("idle-timeout", 120*time.Second, "HTTP server idle timeout")
	chunkSize := flag.Int("chunk-size", 64*1024, "size of each data chunk in bytes")
	randomize := flag.Bool("random", false, "fill download chunks with random data")
	defaultSize := flag.Int("default-size", 10*1024*1024, "default download size in bytes if not specified")
	flag.Parse()

	// Prepare payload chunk once
	payloadChunk = make([]byte, *chunkSize)
	if *randomize {
		if _, err := rand.Read(payloadChunk); err != nil {
			log.Fatalf("Failed to generate random payload chunk: %v", err)
		}
	} else {
		// leave zeros, which is faster
	}

	// Setup mux with logging middleware
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", pingHandler)
	mux.HandleFunc("/download", downloadHandler)
	mux.HandleFunc("/upload", uploadHandler)
	loggedMux := loggingMiddleware(mux)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", *port),
		Handler:      loggedMux,
		ReadTimeout:  *rTimeout,
		WriteTimeout: *wTimeout,
		IdleTimeout:  *idleTimeout,
	}

	log.Printf("Starting speedtest backend: port=%d, chunk-size=%d, random=%v, default-size=%d", *port, *chunkSize, *randomize, *defaultSize)

	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
