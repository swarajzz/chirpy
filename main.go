package main

import (
	"chirpy/internal/database"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var port string = "8080"

type apiConfig struct {
	fileserverHits atomic.Int32
	DB             *database.Queries
}

func handlerReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) handlerMetris(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `
	<html>
  		<body>
    		<h1>Welcome, Chirpy Admin</h1>
    		<p>Chirpy has been visited %d times!</p>
  		</body>
	</html>
`, cfg.fileserverHits.Load())
}

func (apiCfg *apiConfig) resetMetrics(w http.ResponseWriter, r *http.Request) {
	platform := os.Getenv("PLATFORM")
	if platform != "DEV" {
		respondWithError(w, http.StatusForbidden, "On;y allowed in DEV environment")
		return
	}

	err := apiCfg.DB.DeleteUsers(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to delete users")
		return
	}
	apiCfg.fileserverHits.Store(0)

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Metrics reset successfully and users deleted.",
	})
}

func censorString(str string) string {
	words := strings.Split(str, " ")

	for i, word := range words {
		switch strings.ToLower(word) {
		case "kerfuffle":
			words[i] = "****"
		case "sharbert":
			words[i] = "****"
		case "fornax":
			words[i] = "****"
		}
	}

	return strings.Join(words, " ")
}

func handlerValidateChirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}
	params := parameters{}

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&params)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Something went wrong"}`))
		return
	}

	if len(params.Body) > 140 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Chirp is too long"}`))
		return
	}
	censoredString := censorString(params.Body)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("cleaned_body: " + censoredString))
}

func (apiCfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 400, fmt.Sprint("error parsing JSON:", err))
		return
	}

	user, err := apiCfg.DB.CreateUser(r.Context(), params.Email)
	if err != nil {
		log.Fatal(err)
	}
	respondWithJSON(w, 201, databaseUserToUser(user))
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("DB_URL environment variable not set")
	}
	dbUrl := os.Getenv("DB_URL")

	db, err := sql.Open("postgres", dbUrl)
	dbQueries := database.New(db)
	if err != nil {
		log.Fatal("can't connect to database", err)
	}
	mux := http.NewServeMux()

	apiCfg := apiConfig{
		DB: dbQueries,
	}

	mux.Handle("/app/", http.StripPrefix("/app/", apiCfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))

	mux.HandleFunc("GET /api/healthz", handlerReadiness)

	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetris)
	mux.HandleFunc("POST /admin/reset", apiCfg.resetMetrics)

	mux.HandleFunc("POST /api/validate_chirp", handlerValidateChirp)

	mux.HandleFunc("POST /api/users", apiCfg.handlerCreateUser)

	srv := http.Server{
		Handler: mux,
		Addr:    ":" + port,
	}

	fmt.Print("dburl" + dbUrl)
	log.Println("Server is starting on :8080", dbUrl)
	log.Fatal(srv.ListenAndServe())
}
