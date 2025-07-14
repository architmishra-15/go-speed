package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	payloadChunk []byte

	defaultDownloadSize int

	pingCounter      = prometheus.NewCounter(prometheus.CounterOpts{Name: "ping_requests_total", Help: "Number of ping requests"})
	downloadByteCnt  = prometheus.NewCounter(prometheus.CounterOpts{Name: "download_bytes_total", Help: "Total bytes served by /download"})
	uploadByteCnt    = prometheus.NewCounter(prometheus.CounterOpts{Name: "upload_bytes_total", Help: "Total bytes received by /upload"})
)

func init() {
	rand.Seed(time.Now().UnixNano())
	prometheus.MustRegister(pingCounter, downloadByteCnt, uploadByteCnt)
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

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	pingCounter.Inc()
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("pong"))
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	sizeParam := r.URL.Query().Get("size")
	var totalSize int
	var err error
	if sizeParam == "" {
		totalSize = defaultDownloadSize
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
	downloadByteCnt.Add(float64(totalSize))
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
	uploadByteCnt.Add(float64(bytesRead))
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

	defaultDownloadSize = *defaultSize

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
	mux.HandleFunc("/healthz", healthHandler)
	mux.Handle("/metrics", promhttp.Handler())
	loggedMux := loggingMiddleware(mux)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", *port),
		Handler:      loggedMux,
		ReadTimeout:  *rTimeout,
		WriteTimeout: *wTimeout,
		IdleTimeout:  *idleTimeout,
	}

	log.Printf("Starting speedtest backend: port=%d, chunk-size=%d, random=%v, default-size=%d", *port, *chunkSize, *randomize, *defaultSize)

	// run server in goroutine for graceful shutdown
	errChan := make(chan error, 1)
	go func() { errChan <- srv.ListenAndServe() }()

	// listen for interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		log.Printf("Received signal %v, shutting down", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Fatalf("Graceful shutdown failed: %v", err)
		}
	case err := <-errChan:
		log.Fatalf("Server failed: %v", err)
	}
}
