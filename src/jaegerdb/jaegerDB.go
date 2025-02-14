package jaegerdb

import (
	"context"
	"net/http"
	// "fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
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

	query := `INSERT INTO users (full_name, email, password, created_at, updated_at)
	VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
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

// TODO: JWT tokens
func GetUserByEmail(conn *pgx.Conn, email string) (*RetrievedUser, error) {
	var user RetrievedUser
	err := conn.QueryRow(context.Background(), "SELECT id, full_name, email FROM users WHERE email = $1", email).Scan(&user.ID, &user.FullName, &user.Email)
	if err != nil {
		log.Printf("Failed to retrieve user %s", email)
		return &RetrievedUser{}, err
	}
	return &user, nil
}

func CheckCredentialsOnLogin(conn *pgx.Conn, email, password string) error {
	var user LoginData
	user.Email = email // assign the email to user
	// get the pw for that email
	err := conn.QueryRow(context.Background(), "SELECT password FROM users where email = $1", email).Scan(&user.Password)
	if err != nil { // if err log and return err
		log.Printf("Failed to find user with %s email address", email)
		return err
	}

	// compare the existing hash with the pw from frontend
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		log.Print("Hash doesn't match the password")
		return err
	}
	return nil
}

// TODO: create a function that turns 0 values into nil
func UpdateUser(conn *pgx.Conn, id int, fullName, email, password string) (UpdatedUserDataDB, error) {
	var updatedUser UpdatedUserDataDB

	err := conn.QueryRow(context.Background(), "UPDATE users SET full_name = COALESCE($1, full_name), email = COALESCE($2, email), password = COALESCE($3, password) WHERE id = $4 RETURNING id, full_name, email, password;;", stringToNil(&fullName), stringToNil(&email), stringToNil(&password), id).Scan(&updatedUser.id, &updatedUser.fullName, &updatedUser.email, &updatedUser.password)
	if err != nil {
		log.Printf("User couldn't be updated: %s", err)
		return UpdatedUserDataDB{}, err
	}
	return updatedUser, nil
}

// helper for converting 0 values to nil
func stringToNil(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	return s
}
