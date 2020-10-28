package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	hashRoutePath     = "/hash"
	statsRoutePath    = "/stats"
	shutdownRoutePath = "/shutdown"
)

// HashService represents the password hashing service implementation
type HashService struct {
	srv             http.Server
	idleConnsClosed chan struct{}
	once            sync.Once
	storage         *HashStorage
	stats           *HashStatsStorage
}

// NewHashService constructs a new instance of the password hashing service
func NewHashService(httpAddr *string) *HashService {
	hashService := &HashService{}
	hashService.srv = http.Server{Addr: *httpAddr}
	hashService.idleConnsClosed = make(chan struct{})
	hashService.storage = NewHashStorage()
	hashService.stats = NewHashStatsStorage()
	return hashService
}

// Grecefully shut down the server
func (s *HashService) initiateShutdown() {
	// We received a shutdown command, shut down. Make sure we call it only once.
	s.once.Do(func() {
		go func() {
			if err := s.srv.Shutdown(context.Background()); err != nil {
				// Error from closing listeners, or context timeout:
				log.Printf("HTTP server Shutdown: %v\n", err)
			}
			close(s.idleConnsClosed)
		}()
	})
}

// Helper structs for returning JSON
type hashIdentifier struct {
	ID uint64 `json:"id"`
}
type hashValue struct {
	Hash string `json:"hash"`
}

// Run executes the password hashing service
func (s *HashService) Run() {
	// The handler for the web service root - always returns StatusNotFound
	homeHandler := func(w http.ResponseWriter, r *http.Request) {
		log.Printf("homeHandler: Not found (%v)\n", r.URL)
		http.Error(w, "Not found", http.StatusNotFound)
	}

	// The handler for the the new password hash creation calls
	hashPostHandler := func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			startTime := time.Now()
			defer s.stats.Update(startTime)
			if r.URL.Path != hashRoutePath {
				log.Printf("hashPostHandler: Not found (%v)\n", r.URL)
				http.Error(w, "Not found", http.StatusNotFound)
				return
			}
			if err := r.ParseForm(); err != nil {
				log.Printf("hashPostHandler: Bad request: %v\n", err)
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}
			pw := r.FormValue("password")
			if pw == "" {
				log.Println("hashPostHandler: Bad request: missing password")
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}
			u := s.storage.AddPassword(pw)
			val := hashIdentifier{ID: u}
			w.Header().Set("Location", hashRoutePath+"/"+strconv.FormatUint(u, 10))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(val)
			break
		default:
			log.Printf("hashPostHandler: Method %v not allowed\n", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			break
		}
	}

	// The handler for the the password hash retrieval calls
	hashGetHandler := func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			parts := strings.Split(r.URL.Path, "/")
			if len(parts) != 3 || parts[0] != "" || "/"+parts[1] != hashRoutePath {
				log.Printf("hashGetHandler: Not found (%v)\n", r.URL)
				http.Error(w, "Not found", http.StatusNotFound)
				return
			}
			u, err := strconv.ParseUint(parts[2], 10, 64)
			if err != nil {
				log.Printf("hashGetHandler: Bad request: %v\n", err)
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}
			hash, ok := s.storage.GetPasswordHash(u)
			if !ok {
				log.Printf("hashGetHandler: Not found (%v)\n", r.URL)
				http.Error(w, "Not found", http.StatusNotFound)
				return
			}
			val := hashValue{Hash: hash}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(val)
			break
		default:
			log.Printf("hashGetHandler: Method %v not allowed\n", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			break
		}
	}

	// The handler for the the statistics retrieval calls
	statsHandler := func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if r.URL.Path != statsRoutePath {
				log.Printf("statsHandler: Not found (%v)\n", r.URL)
				http.Error(w, "Not found", http.StatusNotFound)
				return
			}
			stats := s.stats.GetCurrentStats()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(stats)
			break
		default:
			log.Printf("statsHandler: Method %v not allowed\n", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			break
		}
	}

	// The handler for the the graceful shutdown calls
	shutdownHandler := func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			if r.URL.Path != shutdownRoutePath {
				log.Printf("shutdownHandler: Not found (%v)\n", r.URL)
				http.Error(w, "Not found", http.StatusNotFound)
				return
			}
			s.initiateShutdown()
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
			break
		default:
			log.Printf("shutdownHandler: Method %v not allowed\n", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			break
		}
	}

	// Initialize route handlers
	http.HandleFunc("/", homeHandler)
	http.HandleFunc(hashRoutePath, hashPostHandler)
	http.HandleFunc(hashRoutePath+"/", hashGetHandler)
	http.HandleFunc(statsRoutePath, statsHandler)
	http.HandleFunc(shutdownRoutePath, shutdownHandler)

	// Begin listening for incoming connections
	if err := s.srv.ListenAndServe(); err != http.ErrServerClosed {
		// Error starting or closing listener:
		log.Fatalf("HTTP server ListenAndServe: %v\n", err)
	}

	// Wait for graceful shutdown
	<-s.idleConnsClosed
}
