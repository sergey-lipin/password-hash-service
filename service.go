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
	Srv             http.Server
	IdleConnsClosed chan struct{}
	Once            sync.Once
	Storage         *HashStorage
	Stats           *HashStatsStorage
}

// NewHashService constructs a new instance of the password hashing service
func NewHashService(httpAddr *string) *HashService {
	hashService := &HashService{}
	hashService.Srv = http.Server{Addr: *httpAddr}
	hashService.IdleConnsClosed = make(chan struct{})
	hashService.Storage = NewHashStorage()
	hashService.Stats = NewHashStatsStorage()
	return hashService
}

func (s *HashService) initiateShutdown() {
	s.Once.Do(func() {
		go func() {
			// We received a shutdown command, shut down.
			if err := s.Srv.Shutdown(context.Background()); err != nil {
				// Error from closing listeners, or context timeout:
				log.Printf("HTTP server Shutdown: %v\n", err)
			}
			close(s.IdleConnsClosed)
		}()
	})
}

type hashValue struct {
	Hash string `json:"hash"`
}

// Run executes the password hashing service
func (s *HashService) Run() {
	homeHandler := func(w http.ResponseWriter, r *http.Request) {
		log.Printf("homeHandler: Not found (%v)\n", r.URL)
		http.Error(w, "Not found", http.StatusNotFound)
	}
	hashPostHandler := func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			defer s.Stats.Update(time.Now())
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
			u, err := s.Storage.AddPassword(pw)
			if err != nil {
				log.Printf("hashPostHandler: Internal server error: %v\n", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Location", hashRoutePath+"/"+strconv.FormatUint(u, 10))
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte("Created"))
			break
		default:
			log.Printf("hashPostHandler: Method %v not allowed\n", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			break
		}
	}
	hashGetHandler := func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			parts := strings.Split(r.URL.Path, "/")
			if len(parts) != 2 || "/"+parts[0] != hashRoutePath {
				log.Printf("hashGetHandler: Not found (%v)\n", r.URL)
				http.Error(w, "Not found", http.StatusNotFound)
				return
			}
			u, err := strconv.ParseUint(parts[1], 10, 64)
			if err != nil {
				log.Printf("hashGetHandler: Bad request: %v\n", err)
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}
			hash, ok := s.Storage.GetPasswordHash(u)
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
	statsHandler := func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if r.URL.Path != statsRoutePath {
				log.Printf("statsHandler: Not found (%v)\n", r.URL)
				http.Error(w, "Not found", http.StatusNotFound)
				return
			}
			stats := s.Stats.GetCurrentStats()
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
	if err := s.Srv.ListenAndServe(); err != http.ErrServerClosed {
		// Error starting or closing listener:
		log.Fatalf("HTTP server ListenAndServe: %v\n", err)
	}

	// Wait for graceful shutdown
	<-s.IdleConnsClosed
}
