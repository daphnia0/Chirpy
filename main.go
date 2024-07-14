package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

type apiConfig struct {
	fileserverHits int
}

type parameters struct {
	Body string `json:"body"`
}

type fail_error struct {
	Error string `json:"error"`
}

type valid struct {
	Valid string `json:"cleaned_body"`
}

type DB struct {
	path string
	mux  *sync.RWMutex
}

type DBStructure struct {
	Chirps map[int]Chirp `json:"chirps"`
}

type Chirp struct {
	id   int    `json:"id"`
	body string `json:"body"`
}

func NewDB(path string) (*DB, error) {
	db := &DB{
		path: path,
		mux:  &sync.RWMutex{},
	}

	if err := db.ensureDB(); err != nil {
		return nil, err
	}

	return db, nil
}

func (db *DB) ensureDB() error {
	// Check if the database file exists
	if _, err := os.Stat(db.path); os.IsNotExist(err) {
		// Create the file with an initial empty structure
		initialData := DBStructure{
			Chirps: make(map[int]Chirp),
		}

		// Marshal the initial structure to JSON
		jsonData, err := json.Marshal(initialData)
		if err != nil {
			return err
		}

		// Write JSON data to the file
		if err := os.WriteFile(db.path, jsonData, 0644); err != nil {
			return err
		}
	}

	return nil
}

func marshallingJson(w http.ResponseWriter, respBody interface{}) []byte {
	dat, err := json.Marshal(respBody)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return nil
	}
	return dat
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	respBody := fail_error{Error: msg}
	dat := marshallingJson(w, respBody)
	if dat != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		w.Write(dat)
	}
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	dat := marshallingJson(w, payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
}

func filerChirpy(body string) string {
	splitedLowerBody := strings.Split(body, " ")
	forbiddenWords := map[string]bool{
		"kerfuffle": true,
		"sharbert":  true,
		"fornax":    true,
	}
	listClearedBody := []string{}

	for _, word := range splitedLowerBody {
		if forbiddenWords[strings.ToLower(word)] {
			listClearedBody = append(listClearedBody, "****")
		} else {
			listClearedBody = append(listClearedBody, word)
		}
	}

	clearedBody := strings.Join(listClearedBody, " ")
	return clearedBody
}

func validateChripHandler(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}
	if len(params.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
		return
	}
	clearedBody := filerChirpy(params.Body)
	respondWithJSON(w, 200, valid{Valid: clearedBody})
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits++
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) handleMetrics(w http.ResponseWriter, r *http.Request) {
	response := fmt.Sprintf("<html> <body> <h1>Welcome, Chirpy Admin</h1> <p>Chirpy has been visited %d times!</p></body></html>", cfg.fileserverHits)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(response))
}

func (cfg *apiConfig) handleReset(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits = 0
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func readinessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	apiCfg := apiConfig{fileserverHits: 0}
	mux := http.NewServeMux()

	fileServer := http.FileServer(http.Dir("/root/bootDotDev/Chirpy/"))

	mux.Handle("/app/", http.StripPrefix("/app", apiCfg.middlewareMetricsInc(fileServer)))
	mux.HandleFunc("GET /api/reset", apiCfg.handleReset)
	mux.HandleFunc("GET /admin/metrics", apiCfg.handleMetrics)
	mux.HandleFunc("GET /api/healthz", readinessHandler)
	mux.HandleFunc("POST /api/validate_chirp", validateChripHandler)

	server := &http.Server{
		Addr:    "127.0.0.1:8081",
		Handler: mux, // your ServeMux that you created earlier
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
