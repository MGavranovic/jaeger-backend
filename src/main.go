package main

import (
	"context"
	"encoding/json"
	"strconv"
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
	mux.HandleFunc("/api/users/logout", apiServer.handleLogoutUser)
	mux.HandleFunc("/api/users/current/", apiServer.handleGetLoggedinUser)
	mux.HandleFunc("/api/users/current", apiServer.handleGetCurrentUser) // this is needed for user data on frontend to be up to date
	mux.HandleFunc("/api/users/current/update", apiServer.handleUpdateUserData)
	mux.HandleFunc("/api/notes/create", apiServer.handleCreateNote)
	mux.HandleFunc("/api/notes/", apiServer.handleGetUserNotes)
	mux.HandleFunc("/api/notes/update", apiServer.handleUpdateNote)
	mux.HandleFunc("/api/notes/current/", apiServer.handleGetCurrentNote)
	mux.HandleFunc("/api/notes/delete/", apiServer.handleDeleteNote)

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

type LoginData struct {
	Password string `json:"password"`
}

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

	// TODO: checking the credentials
	var loginData LoginData // getting the user data from login
	if err := json.NewDecoder(r.Body).Decode(&loginData); err != nil {
		http.Error(w, "Failed to decode JSON to login data", http.StatusInternalServerError)
		log.Printf("Error decoding JSON to login data: %s", err)
		return
	}

	// check pw exists
	if loginData.Password == "" {
		log.Print("Password missing in request body")
		http.Error(w, "Password is required", http.StatusBadRequest)
		return
	}

	// checking the credentials
	if err := jaegerdb.CheckCredentialsOnLogin(s.dbConn, email, loginData.Password); err != nil {
		log.Printf("Credentials don't match the db: %s", err)
		http.Error(w, "Failed to compare credentials", http.StatusInternalServerError)
		return
	}

	token, err := jaegerjwt.GenerateJWT(email) // generating the token
	if err != nil {
		log.Printf("Failed to generate token: %s", err)
		http.Error(w, "Failed to generate token", http.StatusUnauthorized)
		return
	}

	jaegerjwt.SetTokenInCookies(w, token) // setting the cookies

	w.WriteHeader(http.StatusOK) // OK response
	log.Print("Users successfully logged in!")
}

func (s *Server) checkAuth(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// getting the cookie from the request body
	cookie, err := r.Cookie("authToken")
	if err != nil {
		log.Printf("Authorization unssuccessful :%s", err)
		http.Error(w, "No cookie - Unauthorized", http.StatusUnauthorized)
		return
	}
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

// TODO: not return OK with an error
func (s *Server) handleLogoutUser(w http.ResponseWriter, r *http.Request) {
	jaegerjwt.DeleteCookie(w)            // delete cookie
	cookie, err := r.Cookie("authToken") // check for cookie to return OK status
	if err != nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Logged out"))
		return
	}

	log.Printf("Cookie value after logout = %s", cookie) // if cookie exists
}

func (s *Server) handleGetLoggedinUser(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	// checking the path
	path := r.URL.Path
	basePath := "/api/users/current/"
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

	// TODO: get user and send the json to frontend on login
	user, err := jaegerdb.GetUserByEmail(s.dbConn, email)
	log.Printf("GetUserByEmail(%s) = user -> %s", email, user) // checking if we have proper data

	jsonUser, err := json.Marshal(user) // marshalling user
	if err != nil {
		log.Printf("Failed marshaling user data to json: %s", err)
		http.Error(w, "Failed marshaling user data to json", http.StatusInternalServerError)
		return
	}
	log.Print(string(jsonUser))
	w.Write(jsonUser) //sending user
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleGetCurrentUser(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	// get the cookie from the request
	cookie, err := r.Cookie("authToken")
	if err != nil {
		log.Printf("Missing auth token: %s", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// validate cookie
	token, err := jaegerjwt.ValidateToken(cookie.Value)
	if err != nil {
		log.Printf("Invalid token: %s", err)
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// get claims from the token
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		log.Print("Token is not valid")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// get email from the claims
	email, ok := claims["email"].(string)
	if !ok {
		log.Print("Email claim is missing or invalid")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := jaegerdb.GetUserByEmail(s.dbConn, email)
	if err != nil {
		log.Printf("Failed retrieving user: %s", err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// marshal the data and send to frontend
	jsonUser, err := json.Marshal(user)
	if err != nil {
		log.Printf("Error marshaling user: %s", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonUser)
}

type UpdatedUserData struct {
	ID       int    `json:"id"`
	FullName string `json:"fullName"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) handleUpdateUserData(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	var updatedUser UpdatedUserData // decode the updates comming from frontend
	if err := json.NewDecoder(r.Body).Decode(&updatedUser); err != nil {
		http.Error(w, "Failed to decode JSON to updated user", http.StatusInternalServerError)
		log.Printf("Error decoding JSON to updated user: %s", err)
		return
	}

	// TODO: update user here (UpdateUser)
	log.Printf("func handleUpdateUserData -> updated user info from that came in:\nID: %d\nFullName: %s\nEmail: %s\nPassword: %s\n", updatedUser.ID, updatedUser.FullName, updatedUser.Email, updatedUser.Password)

	updatedData, err := jaegerdb.UpdateUser(s.dbConn, updatedUser.ID, updatedUser.FullName, updatedUser.Email, updatedUser.Password)
	if err != nil {
		log.Printf("UpdateUser -> couldn't update: %s", err)
		http.Error(w, "Failed updating user data", http.StatusInternalServerError)
		return
	}

	jsonUpdatedData, err := json.Marshal(updatedData)
	if err != nil {
		log.Printf("Failed marshaling updated user data to json: %s", err)
		http.Error(w, "Failed marshaling updated user data to json", http.StatusInternalServerError)
		return
	}

	w.Write(jsonUpdatedData)
	w.WriteHeader(http.StatusOK) // OK response
	log.Print("Users information successfully updated!")
}

// TODO: handlers for notes
type Note struct {
	UUID              string `json:"uuid"`
	CompanyName       string `json:"companyName"`
	Position          string `json:"position"`
	Salary            string `json:"salary"`
	ApplicationStatus string `json:"applicationStatus"`
	AppliedOn         string `json:"appliedOn"`
	Description       string `json:"description"`
	UserId            int    `json:"userId"`
}

func (s *Server) handleCreateNote(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	var noteData Note
	if err := json.NewDecoder(r.Body).Decode(&noteData); err != nil {
		log.Printf("Failed to decode note data: %s", err)
		http.Error(w, "Failed to decode note data", http.StatusInternalServerError)
		return
	}

	log.Printf("\n*****INCOMMING NOTE*****\nfunc handleCreateNote -> note data that came in from the frontend:\nUUID: %s\nCompanyName: %s\nPosition: %s\nSalary: %s\nApplicationStatus: %s\nAppliedOn: %s\nDescription: %s\nUser ID: %d\n*****END Incomming NOTE*****", noteData.UUID, noteData.CompanyName, noteData.Position, noteData.Salary, noteData.ApplicationStatus, noteData.AppliedOn, noteData.Description, noteData.UserId)

	if err := jaegerdb.CreateNote(s.dbConn, noteData.UUID, noteData.CompanyName, noteData.Position, noteData.Salary, noteData.ApplicationStatus, noteData.AppliedOn, noteData.Description, noteData.UserId); err != nil {
		log.Printf("Failed creating Note: %s", err)
		http.Error(w, "Failed creating Note", http.StatusInternalServerError)
	}

	/*
		newNote.uuid, newNote.companyName, newNote.position, newNote.salary, newNote.applicationStatus, newNote.appliedOn, newNote.description, newNote.userId
	*/
	w.WriteHeader(http.StatusOK) // OK response
	log.Print("Note created successfully!")
}

func (s *Server) handleGetUserNotes(w http.ResponseWriter, r *http.Request) {
	// TODO: extract id from url and pass it to getAllUserNotes DB func
	enableCors(&w)

	// checking the path
	path := r.URL.Path
	basePath := "/api/notes/"
	if !strings.HasPrefix(path, basePath) {
		log.Printf("Invalid URL: %s", path)
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	id := strings.TrimPrefix(path, basePath) // extracing the id from the path
	if id == "" {
		log.Printf("ID missing in the url request: %s", id)
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	intId, err := strconv.Atoi(id)
	if err != nil {
		log.Printf("Invalid ID format: %s", id)
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	userNotes, err := jaegerdb.GetAllUserNotes(s.dbConn, intId)
	// DEBUG: notes sent as null
	log.Printf("Notes sent to frontend %s", userNotes)
	if err != nil {
		log.Printf("Failed to retrieve user notes: %s", err)
		http.Error(w, "Failed to retrieve user notes!", http.StatusInternalServerError)
	}

	if userNotes == nil {
		userNotes = []jaegerdb.NoteDB{}
	}

	jsonNotes, err := json.Marshal(userNotes)
	if err != nil {
		log.Printf("Failed to marshal user notes: %s", err)
		http.Error(w, "Failed to marshal user notes!", http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonNotes)
	log.Print("Notes sent to frontend successfully!")
}

type updatedNote struct {
	CompanyName       string `json:"companyName,omitempty"`
	Position          string `json:"position,omitempty"`
	Salary            string `json:"salary,omitempty"`
	ApplicationStatus string `json:"applicationStatus,omitempty"`
	AppliedOn         string `json:"appliedOn,omitempty"`
	Description       string `json:"description,omitempty"`
	UserId            int    `json:"userId"`
	NoteId            int    `json:"noteId"`
}

func (s *Server) handleUpdateNote(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	var updatedNoteData updatedNote
	if err := json.NewDecoder(r.Body).Decode(&updatedNoteData); err != nil {
		log.Printf("Error decoding updated note data")
		http.Error(w, "Failed decoding updated note data", http.StatusInternalServerError)
	}

	log.Printf("\n*****INCOMMING UPDATED NOTE*****\nfunc handleUpdateNote -> updated note data that came in from the frontend:\nCompanyName: %s\nPosition: %s\nSalary: %s\nApplicationStatus: %s\nAppliedOn: %s\nDescription: %s\nUser ID: %d\nNote ID: %d\n*****END Incomming NOTE*****", updatedNoteData.CompanyName, updatedNoteData.Position, updatedNoteData.Salary, updatedNoteData.ApplicationStatus, updatedNoteData.AppliedOn, updatedNoteData.Description, updatedNoteData.UserId, updatedNoteData.NoteId)

	// NOTE: testing
	if err := jaegerdb.UpdateNote(s.dbConn, updatedNoteData.NoteId, updatedNoteData.CompanyName, updatedNoteData.Position, updatedNoteData.Salary, updatedNoteData.ApplicationStatus, updatedNoteData.AppliedOn, updatedNoteData.Description); err != nil {
		log.Printf("Error updating the Note in DB: %s", err)
		http.Error(w, "Error updating the Note in DB", http.StatusInternalServerError)
	} else {
		log.Printf("Note updated successfully")
		w.WriteHeader(http.StatusOK)
	}
}

func (s *Server) handleGetCurrentNote(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	path := r.URL.Path
	basePath := "/api/notes/current/"
	if !strings.HasPrefix(path, basePath) {
		log.Printf("Invalid URL: %s", path)
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	id := strings.TrimPrefix(path, basePath)
	if id == "" {
		log.Printf("ID missing in the url request: %s", id)
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	intId, err := strconv.Atoi(id)
	if err != nil {
		log.Printf("Invalid ID format in getCurrentNote handler: %s", id)
		http.Error(w, "Invalid ID format in getCurrentNote handler", http.StatusBadRequest)
		return
	}

	note := jaegerdb.GetUpdatedNote(s.dbConn, intId)
	jsonUpdatedNote, err := json.Marshal(note)
	if err != nil {
		log.Printf("Failed marshaling updated note data to json: %s", err)
		http.Error(w, "Failed marshaling updated note data to json", http.StatusInternalServerError)
		return
	}
	w.Write(jsonUpdatedNote)
	w.WriteHeader(http.StatusOK)
	log.Print("Note data successfully updated!")
}

func (s *Server) handleDeleteNote(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	path := r.URL.Path
	basePath := "/api/notes/delete/"
	if !strings.HasPrefix(path, basePath) {
		log.Printf("Invalid URL: %s", path)
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	id := strings.TrimPrefix(path, basePath)
	if id == "" {
		log.Printf("ID missing in the url request: %s", id)
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	intId, err := strconv.Atoi(id)
	if err != nil {
		log.Printf("Invalid ID format in handleDeleteNote handler: %s", id)
		http.Error(w, "Invalid ID format in handleDeleteNote handler", http.StatusBadRequest)
		return
	}

	if err := jaegerdb.DeleteNote(s.dbConn, intId); err != nil {
		log.Printf("Issues deleting note from DB: %s", err)
		http.Error(w, "Unable to delete note from DB", http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
}
