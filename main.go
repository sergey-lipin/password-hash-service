package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
)

type hashStats struct {
	Total   int `json:"total"`
	Average int `json:"average"`
}

func main() {
	shutdownCalled := make(chan struct{})
	var once sync.Once

	homeHandler := func(w http.ResponseWriter, r *http.Request) {
		log.Println("HOME: Incoming Request:", r.Method)
		log.Println("HOME: Not found")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not found"))
	}
	hashPostHandler := func(w http.ResponseWriter, r *http.Request) {
		log.Println("HASH POST: Incoming Request:", r.Method)
		switch r.Method {
		case http.MethodPost:
			break
		default:
			log.Println("HASH POST: Method not allowed")
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte("Method not allowed"))
			break
		}
	}
	hashGetHandler := func(w http.ResponseWriter, r *http.Request) {
		log.Println("HASH GET: Incoming Request:", r.Method)
		switch r.Method {
		case http.MethodGet:
			break
		default:
			log.Println("HASH GET: Method not allowed")
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte("Method not allowed"))
			break
		}
	}
	statsHandler := func(w http.ResponseWriter, r *http.Request) {
		log.Println("STATS: Incoming Request:", r.Method)
		switch r.Method {
		case http.MethodGet:
			stats := hashStats{Total: 0, Average: 0}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(stats)
			break
		default:
			log.Println("STATS: Method not allowed")
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte("Method not allowed"))
			break
		}
	}
	shutdownHandler := func(w http.ResponseWriter, r *http.Request) {
		log.Println("SHUTDOWN: Incoming Request:", r.Method)
		switch r.Method {
		case http.MethodPost:
			once.Do(func() { close(shutdownCalled) })
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
			break
		default:
			log.Println("SHUTDOWN: Method not allowed")
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte("Method not allowed"))
			break
		}
	}

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/hash", hashPostHandler)
	http.HandleFunc("/hash/", hashGetHandler)
	http.HandleFunc("/stats", statsHandler)
	http.HandleFunc("/shutdown", shutdownHandler)

	srv := http.Server{Addr: ":8080"}

	idleConnsClosed := make(chan struct{})
	go func() {
		<-shutdownCalled

		// We received a shutdown command, shut down.
		if err := srv.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			log.Printf("HTTP server Shutdown: %v", err)
		}
		close(idleConnsClosed)
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		// Error starting or closing listener:
		log.Fatalf("HTTP server ListenAndServe: %v", err)
	}

	<-idleConnsClosed
}
