package main

import (
	"context"
	// "fmt"
	"log"
	"net/http"
	"os"

	"github.com/MGavranovic/jaeger-backend/src/jaegerdb"
)

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

	// Testing DB package
	dbConn := jaegerdb.ConnectJaegerDB()
	defer dbConn.Close(context.Background())

	// TODO: create internal server package
	// Server setup
	mux := http.NewServeMux() // creating servemux
	server := http.Server{    // server config
		Addr:    ":8080",
		Handler: mux,
	}

	// Server starting
	log.Print("Server starting on port 8080")
	if err := server.ListenAndServe(); err != nil { // ListenAndServe will block execution
		log.Printf("Error starting the server: %s", err)
	}
}
