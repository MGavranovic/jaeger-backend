package jaegerdb

import "time"

type UsersDB struct {
	id        string
	fullName  string
	email     string
	password  string
	createdAt time.Time
	updatedAt time.Time
}

type RetrievedUser struct {
	ID       int
	FullName string
	Email    string
}

type LoginData struct {
	Email    string
	Password string
}

type UpdatedUserDataDB struct {
	id       int
	fullName string
	email    string
	password string
}
