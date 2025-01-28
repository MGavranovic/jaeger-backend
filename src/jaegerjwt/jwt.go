package jaegerjwt

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	// "time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
)

// NOTE: check the server.log and make sure it's not uploaded anywhere prior to that

func GenerateJWT(email string) (string, error) {
	if err := godotenv.Load("../.env"); err != nil {
		log.Printf("Failed loading the environment file: %s", err)
	}

	jwtKey := []byte(os.Getenv("JWT_KEY"))

	claims := jwt.MapClaims{
		"email": email,
		"exp":   time.Now().Add(time.Hour * 24).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

func SetTokenInCookies(w http.ResponseWriter, token string) {
	// duration := time.Now().Add(time.Hour * 24)

	cookie := &http.Cookie{
		Name:     "authToken",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
		// Expires:  duration,
	}
	http.SetCookie(w, cookie)
}

func DeleteCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     "authToken",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
		MaxAge:   -1,
	}
	http.SetCookie(w, cookie)
}

func ValidateToken(tokenStr string) (*jwt.Token, error) {
	if err := godotenv.Load("../.env"); err != nil {
		log.Printf("Failed loading the environment file: %s", err)
	}
	jwtKey := []byte(os.Getenv("JWT_KEY"))
	// Parse the token with the secret key
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		// Ensure that the signing method is correct (e.g., HS256)
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		// Return the secret key for verification
		return jwtKey, nil
	})

	if err != nil {
		return nil, err
	}
	// Return the token if it's valid
	return token, nil
}
