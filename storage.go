package main

import (
	"crypto/sha512"
	"encoding/base64"
	"sync"
)

// HashStorage represents the password hash storage implementation
type HashStorage struct {
	mu         sync.RWMutex
	data       map[uint64]string
	currentKey uint64
}

// NewHashStorage constructs a new instance of the password hash storage
func NewHashStorage() *HashStorage {
	hashStorage := &HashStorage{data: make(map[uint64]string)}
	return hashStorage
}

// AddPassword adds a new pasword hash record to the storage and returns its identifier
func (s *HashStorage) AddPassword(pw string) (u uint64, err error) {
	alg := sha512.New()
	_, err = alg.Write([]byte(pw))
	if err != nil {
		return
	}
	encodedHash := base64.StdEncoding.EncodeToString(alg.Sum(nil))
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentKey++
	s.data[s.currentKey] = encodedHash
	u = s.currentKey
	return
}

// GetPasswordHash returns the previously stored hash
func (s *HashStorage) GetPasswordHash(u uint64) (encodedHash string, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	encodedHash, ok = s.data[u]
	return
}
