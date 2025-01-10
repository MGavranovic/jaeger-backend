package jaegerdb

import (
	"context"
	"net/http"
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

func CreateUserJaeger(w http.ResponseWriter, conn *pgx.Conn, fullName, email, password string) error { // return an int to be able to differentiate between server and insert error
	tx, err := conn.Begin(context.Background()) // begin the transaction
	if err != nil {
		log.Printf("Unable to start DB transaction: %s", err)
		return err
	}
	defer func() { // will run when the surrounding code is executed
		if err != nil {
			tx.Rollback(context.Background()) // rollback if an issue comes up
		} else {
			tx.Commit(context.Background()) // commiting the transaction
		}
	}()

	query := `INSERT INTO users (full_name, email, password, created_at, updated_at, is_authenticated)
	VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, False)
	ON CONFLICT (email) DO NOTHING;` // query for inserting the user

	cmdTag, err := tx.Exec(context.Background(), query, fullName, email, password) // executing the query
	if err != nil {
		log.Printf("Unable to insert the data into DB: %s", err)
		return err
	}

	if cmdTag.RowsAffected() == 0 { // error on unaffected rows
		log.Printf("No rows inserted. A user with email %s already exists.", email)
		http.Error(w, "No rows inserted. A user with this email already exists.", http.StatusConflict)
		return nil
	}
	return nil
}

// TODO: change is_authenticated to True during signup where login is happening
func GetUserByEmail(conn *pgx.Conn, email string) (*RetrievedUser, error) {
	var user RetrievedUser
	err := conn.QueryRow(context.Background(), "SELECT id, full_name, email, is_authenticated FROM users WHERE email = $1", email).Scan(&user.ID, &user.FullName, &user.Email, &user.IsAuthenticated)
	if err != nil {
		log.Printf("Failed to retrieve user %s", email)
		return &RetrievedUser{}, err
	}
	return &user, nil
}
