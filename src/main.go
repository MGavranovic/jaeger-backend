package main

import (
	"context"
	"encoding/json"
	"strings"

	// "fmt"
	"log"
	"net/http"
	"os"

	"github.com/MGavranovic/jaeger-backend/src/jaegerdb"
	"github.com/MGavranovic/jaeger-backend/src/jaegerjwt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
)

type Server struct {
	dbConn *pgx.Conn
}

func main() {
	// TODO: create internal server logging
	// Creating a log file
	file, err := os.OpenFile("../logs/server.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open the log file: %s", err)
	}
	defer file.Close() // Closing the log file

	log.SetOutput(file)         // Setting the outpot to the log file
	log.Printf("Start logging") // Test printing to the same logfile

	// Connecting to DB
	dbConn := jaegerdb.ConnectJaegerDB()
	defer dbConn.Close(context.Background())

	apiServer := &Server{
		dbConn: dbConn,
	}

	// TODO: create internal server package
	// Server setup
	mux := http.NewServeMux()  // creating servemux
	httpServer := http.Server{ // server config
		Addr:    ":8080",
		Handler: mux,
	}

	// API endpoints
	mux.HandleFunc("/api/users", apiServer.handleGetUsers)
	mux.HandleFunc("/api/users/signup", apiServer.handleSignupUser)
	mux.HandleFunc("/api/users/login/", apiServer.handleLoginUser)
	mux.HandleFunc("/api/users/login/auth", apiServer.checkAuth)

	// Server starting
	log.Print("Server starting on port 8080")
	if err := httpServer.ListenAndServe(); err != nil { // ListenAndServe will block execution
		log.Printf("Error starting the server: %s", err)
	}
}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
	(*w).Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	(*w).Header().Set("Access-Control-Allow-Headers", "*")
	(*w).Header().Set("Access-Control-Allow-Credentials", "true")
}

func (s *Server) handleGetUsers(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	users := jaegerdb.GetUsersJaeger(s.dbConn) // getting users from db

	// Marshalling users to JSON
	data, err := json.Marshal(users)
	if err != nil {
		http.Error(w, "Failed to encode users to JSON", http.StatusInternalServerError)
		log.Printf("Error encoding users to JSON: %s", err)
		return
	}
	w.Header().Set("Content-Type", "application/json") // setting response header to JSON
	w.Write(data)                                      // sending data
}

type UserFromFrontend struct {
	FullName string `json:"fullName"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) handleSignupUser(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	var newUser UserFromFrontend
	// decoding new user coming from frontend
	if err := json.NewDecoder(r.Body).Decode(&newUser); err != nil {
		http.Error(w, "Failed to decode JSON to new user", http.StatusInternalServerError)
		log.Printf("Error decoding JSON to new user: %s", err)
		return
	}

	// adding the new user to the DB
	if err := jaegerdb.CreateUserJaeger(w, s.dbConn, newUser.FullName, newUser.Email, newUser.Password); err != nil {
		log.Printf("Error creating the new user and adding it to DB: %s", err)
		http.Error(w, "Failed to add new user to DB", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("User created successfully"))

}

// TODO: handleGetCurrentUser and call it in signup
func (s *Server) handleLoginUser(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	// checking the path
	path := r.URL.Path
	basePath := "/api/users/login/"
	if !strings.HasPrefix(path, basePath) {
		log.Printf("Invalid URL: %s", path)
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	email := strings.TrimPrefix(path, basePath) // extracing the email from the path
	if email == "" {
		log.Printf("Email missing in the url request: %s", email)
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}

	token, err := jaegerjwt.GenerateJWT(email) // generating the token
	if err != nil {
		log.Printf("Failed to generate token: %s", err)
		http.Error(w, "Failed to generate token", http.StatusUnauthorized)
		return
	}

	// TODO: checking the credentials
	jaegerjwt.SetTokenInCookies(w, token) // setting the cookies

	w.WriteHeader(http.StatusOK) // OK response
	log.Print("Users successfully logged in!")
}

func (s *Server) checkAuth(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	// NOTE:  debugging
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	for _, cookie := range r.Cookies() {
		log.Printf("Receieved cooke: %s = %s", cookie.Name, cookie.Value)
	}
	// getting the cookie from the request body
	cookie, err := r.Cookie("authToken")
	// DEBUG: cookie seems to be unavailable here
	log.Printf("cookie %s", cookie)
	if err != nil {
		log.Printf("Authorization unssuccessful :%s", err)
		http.Error(w, "No cookie - Unauthorized", http.StatusUnauthorized)
		return
	}
	// NOTE:  debugging
	log.Printf("Cookie retrieved: %s = %s", cookie.Name, cookie.Value)

	// validating the token from the cookie
	token, err := jaegerjwt.ValidateToken(cookie.Value)
	if err != nil {
		log.Printf("Invalid token: %s", err)
		http.Error(w, "Invalid token", http.StatusUnauthorized)
	}

	// extracting claims
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userID := claims["email"].(string)
		log.Printf("User %s is authenticated", userID)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Authenticated"))
	} else {
		log.Print("User is not authenticated")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}
}
