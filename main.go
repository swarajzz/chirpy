package main

import (
	"chirpy/internal/auth"
	"chirpy/internal/database"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var port string = "8080"

type apiConfig struct {
	fileserverHits atomic.Int32
	DB             *database.Queries
	Secret         string
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
		respondWithError(w, http.StatusForbidden, "Only allowed in DEV environment")
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

func (apiCfg *apiConfig) handlerValidateChirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}
	params := parameters{}

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&params)
	if err != nil {
		respondWithJSON(w, http.StatusInternalServerError, []byte(`{"error": "Something went wrong"}`))
	}

	if len(params.Body) > 140 {
		respondWithJSON(w, http.StatusBadRequest, []byte(`{"error": "Chirp is too long"}`))
	}
	censoredString := censorString(params.Body)

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, `"error": "Authorization token is missing or invalid"`)
		return
	}

	userId, err := auth.ValidateJWT(token, apiCfg.Secret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid or expired token")
		return
	}

	chirp, err := apiCfg.DB.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   censoredString,
		UserID: userId,
	})
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Failed to create chirp")
		return
	}
	respondWithJSON(w, http.StatusCreated, databaseChirpToChirp(chirp))
}

func (cfg *apiConfig) handlerGetChirps(w http.ResponseWriter, r *http.Request) {
	if chirps, ok := cfg.DB.GetChirps(r.Context()); ok == nil {
		respondWithJSON(w, http.StatusOK, databaseChirpsToChirps(chirps))
	} else {
		respondWithError(w, http.StatusInternalServerError, ok.Error())
		return
	}
}

func (cfg *apiConfig) handlerGetChirp(w http.ResponseWriter, r *http.Request) {
	chirpID := r.PathValue("chirpID")

	parsedChirpID, err := uuid.Parse(chirpID)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	if chirp, ok := cfg.DB.GetChirp(r.Context(), parsedChirpID); ok == nil {
		respondWithJSON(w, http.StatusOK, databaseChirpToChirp(chirp))
	} else {
		respondWithError(w, http.StatusNotFound, ok.Error())
		return
	}
}

func (apiCfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Error parsing request body: %v", err))
		return
	}

	hashed_password, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error processing password")
		return
	}

	user, err := apiCfg.DB.CreateUser(r.Context(), database.CreateUserParams{
		Email:          params.Email,
		HashedPassword: hashed_password,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating user")
		return
	}
	respondWithJSON(w, 201, databaseUserToUser(user))
}

func (apiCfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)

	if err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprint("error parsing JSON:", err))
		return
	}

	if params.Email == "" || params.Password == "" {
		respondWithError(w, http.StatusBadRequest, "Email and password are required")
		return
	}

	user, err := apiCfg.DB.GetUserFromEmail(r.Context(), params.Email)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error retrieving user")
		return
	}

	if err := auth.CheckPasswordHash(params.Password, user.HashedPassword); err == nil {
		accessToken, err := auth.MakeJWT(user.ID, apiCfg.Secret)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Error creating JWT token")
			return
		}

		refresh_token, err := auth.MakeRefreshToken()
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		apiCfg.DB.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
			Token:     refresh_token,
			UserID:    user.ID,
			ExpiresAt: time.Now().Add(time.Hour * 24 * 60),
			RevokedAt: sql.NullTime{Valid: false},
		})

		respondWithJSON(w, 200, databaseUserWithAuth(user, accessToken, refresh_token))
	} else {
		fmt.Println(err)
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
}

func (apiCfg *apiConfig) handlerRefresh(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, `"error": "Authorization token is missing or invalid"`)
		return
	}

	user, err := apiCfg.DB.GetUserFromRefreshToken(r.Context(), token)

	if err != nil {
		respondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	accessToken, err := auth.MakeJWT(user.ID, apiCfg.Secret)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating JWT token")
		return
	}

	respondWithJSON(w, 200, map[string]string{
		"token": accessToken,
	})
}

func (apiCfg *apiConfig) handlerRevoke(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, `"error": "Authorization token is missing or invalid"`)
		return
	}

	err = apiCfg.DB.RevokeToken(r.Context(), token)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (apiCfg *apiConfig) handlerUpdateCredentials(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	params := parameters{}

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&params)
	if err != nil {
		respondWithJSON(w, http.StatusInternalServerError, []byte(`{"error": "Something went wrong"}`))
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, `"error": "Authorization token is missing or invalid"`)
		return
	}

	userId, err := auth.ValidateJWT(token, apiCfg.Secret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid or expired token")
		return
	}

	hashed_password, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error processing password")
		return
	}

	user, err := apiCfg.DB.UpdateUserCredentials(r.Context(), database.UpdateUserCredentialsParams{
		ID:             userId,
		Email:          params.Email,
		HashedPassword: hashed_password,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, databaseUserToUser(user))
}

func (apiCfg *apiConfig) handlerDeleteChirp(w http.ResponseWriter, r *http.Request) {
	chirpId := r.PathValue("chirpID")

	parsedChirpId, err := uuid.Parse(chirpId)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, `"error": "Authorization token is missing or invalid"`)
		return
	}

	_, err = auth.ValidateJWT(token, apiCfg.Secret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid or expired token")
		return
	}

	_, err = apiCfg.DB.GetChirp(r.Context(), parsedChirpId)
	if err != nil {
		respondWithError(w, http.StatusNotFound, err.Error())
		return
	}

	err = apiCfg.DB.DeleteChirp(r.Context(), parsedChirpId)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusNoContent, nil)
}

func (apiCfg *apiConfig) handlerPolkaWebhook(w http.ResponseWriter, r *http.Request) {
	type UserData struct {
		UserId uuid.UUID `json:"user_id"`
	}
	type parameters struct {
		Event string   `json:"event"`
		Data  UserData `json:"data"`
	}

	params := parameters{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&params); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if params.Event == "user.upgraded" {
		user, err := apiCfg.DB.GetUserFromId(r.Context(), params.Data.UserId)
		if err != nil {
			respondWithError(w, http.StatusNotFound, "User not found")
			return
		}

		apiCfg.DB.UpgradeUser(r.Context(), user.ID)
		respondWithJSON(w, http.StatusNoContent, nil)
	} else {
		respondWithJSON(w, http.StatusNoContent, nil)
	}
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

	secret := os.Getenv("SECRET")
	apiCfg := apiConfig{
		DB:     dbQueries,
		Secret: secret,
	}

	mux.Handle("/app/", http.StripPrefix("/app/", apiCfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))

	mux.HandleFunc("GET /api/healthz", handlerReadiness)

	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetris)
	mux.HandleFunc("POST /admin/reset", apiCfg.resetMetrics)

	mux.HandleFunc("POST /api/login", apiCfg.handlerLogin)
	mux.HandleFunc("POST /api/users", apiCfg.handlerCreateUser)

	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.handlerGetChirp)
	mux.HandleFunc("GET /api/chirps", apiCfg.handlerGetChirps)
	mux.HandleFunc("POST /api/chirps", apiCfg.handlerValidateChirp)

	mux.HandleFunc("POST /api/refresh", apiCfg.handlerRefresh)
	mux.HandleFunc("POST /api/revoke", apiCfg.handlerRevoke)

	mux.HandleFunc("PUT /api/users", apiCfg.handlerUpdateCredentials)
	mux.HandleFunc("DELETE /api/chirps/{chirpID}", apiCfg.handlerDeleteChirp)

	mux.HandleFunc("POST /api/polka/webhooks", apiCfg.handlerPolkaWebhook)

	srv := http.Server{
		Handler: mux,
		Addr:    ":" + port,
	}

	fmt.Print("dburl" + dbUrl)
	log.Println("Server is starting on :8080", dbUrl)
	log.Fatal(srv.ListenAndServe())
}
