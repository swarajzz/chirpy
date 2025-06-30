package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hashedPassword), nil
}

func CheckPasswordHash(password, hash string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		return err
	}

	return nil
}

func getClaims(userID uuid.UUID) *jwt.RegisteredClaims {
	const defaultExpiration = time.Hour

	claims := &jwt.RegisteredClaims{
		Issuer:    "chirpy",
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		Subject:   userID.String(),
	}

	return claims
}

func MakeJWT(userID uuid.UUID, tokenSecret string) (string, error) {
	claims := getClaims(userID)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	fmt.Println("TOken", tokenSecret)

	tokenString, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", nil
	}

	return tokenString, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(tokenSecret), nil
	})
	if err != nil || !token.Valid {
		return uuid.Nil, err
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, err
	}

	return userID, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	header := headers.Get("Authorization")

	if header == "" {
		return "", errors.New("no bearer token provided")
	}

	if !strings.HasPrefix(header, "bearer ") {
		return "", errors.New("invalid bearer token format")
	}

	token := strings.TrimSpace(header[len("bearer "):])

	return token, nil
}

func MakeRefreshToken() (string, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	token := hex.EncodeToString(key)
	return token, nil
}
