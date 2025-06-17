package main

import (
	"log"
	"net/http"
)

var port string = "8080"

func handlerReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	mux := http.NewServeMux()

	mux.Handle("/app/", http.StripPrefix("/app/", http.FileServer(http.Dir("."))))

	mux.HandleFunc("/healthz", handlerReadiness)

	srv := http.Server{
		Handler: mux,
		Addr:    ":" + port,
	}

	log.Println("Server is starting on :8080")

	log.Fatal(srv.ListenAndServe())
}
