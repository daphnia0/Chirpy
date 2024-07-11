package main

import (
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    "127.0.0.1:8081",
		Handler: mux, // your ServeMux that you created earlier
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
