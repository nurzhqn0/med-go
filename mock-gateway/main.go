package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"med-go/internal/platform/bootstrap"
)

type notifyRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
	Channel        string `json:"channel"`
	Recipient      string `json:"recipient"`
	Message        string `json:"message"`
}

type server struct {
	mu   sync.Mutex
	seen map[string]struct{}
}

func main() {
	bootstrap.LoadDotEnv(".env")
	config := bootstrap.LoadConfig()

	s := &server{seen: make(map[string]struct{})}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /notify", s.notify)

	addr := ":" + config.GatewayPort
	log.Printf("mock-gateway listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("mock-gateway exited with error: %v", err)
	}
}

func (s *server) notify(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var request notifyRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	logRequest(request)

	s.mu.Lock()
	_, duplicate := s.seen[request.IdempotencyKey]
	if duplicate {
		s.mu.Unlock()
		writeJSON(w, http.StatusOK, map[string]string{"status": "duplicate"})
		return
	}

	if rand.Intn(100) < 20 {
		s.mu.Unlock()
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
		return
	}

	s.seen[request.IdempotencyKey] = struct{}{}
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
}

func logRequest(request notifyRequest) {
	line := struct {
		Time string        `json:"time"`
		Path string        `json:"path"`
		Body notifyRequest `json:"body"`
	}{
		Time: time.Now().UTC().Format(time.RFC3339),
		Path: "/notify",
		Body: request,
	}

	if err := json.NewEncoder(os.Stdout).Encode(line); err != nil {
		log.Printf("failed to write gateway log: %v", err)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("failed to write response: %v", err)
	}
}
