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

type Server struct {
	dbConn *pgx.Conn
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

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
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

	// Server starting
	log.Print("Server starting on port 8080")
	if err := httpServer.ListenAndServe(); err != nil { // ListenAndServe will block execution
		log.Printf("Error starting the server: %s", err)
	}
}
