package main

import (
	"context"
	"encoding/json"
	// "fmt"
	"log"
	"net/http"
	"os"

	"github.com/MGavranovic/jaeger-backend/src/jaegerdb"
	"github.com/jackc/pgx/v5"
)

func handleGetUsers(w http.ResponseWriter, r *http.Request, conn *pgx.Conn) {
	users := jaegerdb.GetUsersJaeger(conn) // getting users from db

	w.Header().Set("Content-Type", "application/json") // setting response header to JSON

	// Encoding users to JSON
	if err := json.NewEncoder(w).Encode(users); err != nil {
		http.Error(w, "Failed to encode users to JSON", http.StatusInternalServerError)
		log.Printf("Error encoding users to JSON: %s", err)
		return
	}
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

	// TODO: create internal server package
	// Server setup
	mux := http.NewServeMux() // creating servemux
	server := http.Server{    // server config
		Addr:    ":8080",
		Handler: mux,
	}

	// API endpoints
	mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) { handleGetUsers(w, r, dbConn) })

	// Server starting
	log.Print("Server starting on port 8080")
	if err := server.ListenAndServe(); err != nil { // ListenAndServe will block execution
		log.Printf("Error starting the server: %s", err)
	}
}
