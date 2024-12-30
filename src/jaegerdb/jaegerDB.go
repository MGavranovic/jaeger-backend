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

func CreateUserJaeger(conn *pgx.Conn, fullName, email, password string) error {
	tx, err := conn.Begin(context.Background()) // begin the transaction
	if err != nil {
		log.Printf("Unable to start DB transaction: %s", err)
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback(context.Background()) // rollback if an issue comes up
		} else {
			tx.Commit(context.Background()) // commiting the transaction
		}
	}()

	query := `INSERT INTO users(full_name, email, password, created_at, updated_at) VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);` // query for inserting the user

	_, err = tx.Exec(context.Background(), query, fullName, email, password) // executing the query
	if err != nil {
		log.Printf("Unable to insert the data into DB: %s", err)
		return err
	}
	return nil
}
