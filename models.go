package main

import (
	"chirpy/internal/database"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID    `json:"id"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	Email       string       `json:"email"`
	IsChirpyRed sql.NullBool `json:"is_chirpy_red"`
}

type AuthResponse struct {
	User
	AccessToken  string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

func databaseUserToUser(dbUser database.User) User {
	user := User{
		ID:          dbUser.ID,
		CreatedAt:   dbUser.CreatedAt,
		UpdatedAt:   dbUser.UpdatedAt,
		Email:       dbUser.Email,
		IsChirpyRed: dbUser.IsChirpyRed,
	}
	return user
}

func databaseUserWithAuth(dbUser database.User, tokens ...string) AuthResponse {
	authResponse := AuthResponse{
		User: databaseUserToUser(dbUser),
	}

	if len(tokens) > 0 {
		authResponse.AccessToken = tokens[0]
	}

	if len(tokens) > 1 {
		authResponse.RefreshToken = tokens[1]
	} else if len(tokens) == 1 {
		authResponse.RefreshToken = tokens[0]
	}

	return authResponse
}

func databaseChirpToChirp(dbChirp database.Chirp) Chirp {
	return Chirp{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		UserID:    dbChirp.UserID,
	}
}

func databaseChirpsToChirps(dbChirp []database.Chirp) []Chirp {
	chirps := []Chirp{}
	for _, dbChirp := range dbChirp {
		chirps = append(chirps, databaseChirpToChirp(dbChirp))
	}
	return chirps
}
