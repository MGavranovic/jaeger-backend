package jaegerdb

import (
	"context"
	// "fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func ConnectJaegerDB() *pgx.Conn {
	if err := godotenv.Load("../.env"); err != nil {
		log.Printf("Failed loading the environment file: %s", err)
	}

	conn, err := pgx.Connect(context.Background(), os.Getenv("DB_URL"))
	if err != nil {
		log.Printf("Unable to establish the connection to the DB: %s", err)
	}
	log.Print("Connection has been established with the DB")

	return conn
}

func GetUsersJaeger(conn *pgx.Conn) pgx.Row {
	users, err := conn.Query(context.Background(), "select * from users;")
	if err != nil {
		log.Printf("Unable to retrieve the users from the DB: %s", err)
	}
	defer users.Close()
	return users
}
