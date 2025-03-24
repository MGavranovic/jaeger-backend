package jaegerdb

import (
	"context"
	"fmt"
	"net/http"
	"time"

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

func CreateNote(conn *pgx.Conn, uuid, companyName, position, salary, applicationStatus, appliedOn, description string, userId int) error {

	_, err := conn.Exec(context.Background(), `INSERT INTO notes(
	note_id, company_name, "position", salary, application_status, applied_on, description, updated_at, fk_user_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP, $8);`, uuid, companyName, position, salary, applicationStatus, appliedOn, description, userId)

	if err != nil {
		return err
	}
	return nil
}

func GetAllUserNotes(conn *pgx.Conn, id int) ([]NoteDB, error) {
	rows, err := conn.Query(context.Background(), `SELECT * FROM notes WHERE fk_user_id = $1`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []NoteDB
	for rows.Next() {
		var note NoteDB
		var appliedOn time.Time
		var updatedAt time.Time

		if err := rows.Scan(&note.Id, &note.Uuid, &note.CompanyName, &note.Position, &note.Salary, &note.ApplicationStatus, &appliedOn, &note.UserId, &updatedAt, &note.Description); err != nil {
			return nil, err
		}
		note.AppliedOn = appliedOn.Format("2006-01-02 15:04:05")
		note.UpdatedAt = updatedAt.Format("2006-01-02 15:04:05")

		notes = append(notes, note)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return notes, nil
}

func getNote(conn *pgx.Conn, id int) CheckNoteForUpdate {
	var note CheckNoteForUpdate
	if err := conn.QueryRow(context.Background(), `SELECT company_name, position, salary, application_status, applied_on, description FROM notes WHERE id = $1`, id).Scan(
		&note.companyName, &note.position, &note.salary, &note.status, &note.appliedOn, &note.description); err != nil {
		log.Printf("Error getting the note for updating")
	}

	return note
}

func UpdateNote(conn *pgx.Conn, id int, company, pos, sal, appStat, appOn, desc string) {
	query := "UPDATE notes SET" // query to append to
	// 1. get the data for this note
	existingData := getNote(conn, id)
	log.Printf("EXISTING NOTE DATA: \nCompany Name: %s\nPosition: %s\nSalary: %s\nApplication Status: %s\nApplied On: %s\nDescription: %s\n", existingData.companyName, existingData.position, existingData.salary, existingData.status, existingData.appliedOn, existingData.description)
	// 2. compare it with new data
	args := []any{}
	counter := 1
	if existingData.companyName != company {
		query += fmt.Sprintf(" company_name = $%d,", counter)
		args = append(args, company)
		counter++
	}
	if existingData.position != pos {
		query += fmt.Sprintf(" position = $%d,", counter)
		args = append(args, pos)
		counter++
	}
	if existingData.salary != sal {
		query += fmt.Sprintf(" salary = $%d,", counter)
		args = append(args, sal)
		counter++
	}
	if existingData.status != appStat {
		query += fmt.Sprintf(" application_status = $%d,", counter)
		args = append(args, appStat)
		counter++
	}

	appliedOnDate, err := time.Parse("2006-01-02", appOn)
	if err != nil {
		log.Printf("There is an error with parsing the date: %s", err)
	}
	now := time.Now()

	fullDateTime := time.Date(appliedOnDate.Year(), appliedOnDate.Month(), appliedOnDate.Day(), now.Hour(), now.Minute(), now.Second(), 0, time.UTC)
	existingAppliedOn := existingData.appliedOn.UTC()

	if !existingAppliedOn.Equal(fullDateTime) {
		query += fmt.Sprintf(" applied_on = $%d,", counter)
		args = append(args, fullDateTime)
		counter++
	}
	if existingData.description != desc {
		query += fmt.Sprintf(" description = $%d,", counter)
		args = append(args, desc)
		counter++
	}

	log.Print("*************************args*********************************\n", args, "\n", "*************************args*********************************\n")

}

/*
CompanyName       string `json:"companyName,omitempty"`
	Position          string `json:"position,omitempty"`
	Salary            string `json:"salary,omitempty"`
	ApplicationStatus string `json:"applicationStatus,omitempty"`
	AppliedOn         string `json:"appliedOn,omitempty"`
	Description       string `json:"description,omitempty"`
	UserId            int    `json:"userId"`
	NoteId            int    `json:"noteId"
*/
