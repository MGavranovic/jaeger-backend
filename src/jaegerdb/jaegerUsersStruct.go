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
	fullName *string
	email    *string
	password *string
}

// structs for notes
type NewNote struct {
	uuid              string
	companyName       string
	position          string
	salary            string
	applicationStatus string
	appliedOn         string
	description       string
	userId            int
}
