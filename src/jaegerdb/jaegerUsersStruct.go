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
type NoteDB struct {
	Id                int    `json:"id"`
	Uuid              string `json:"uuid"`
	CompanyName       string `json:"companyName"`
	Position          string `json:"position"`
	Salary            string `json:"salary"`
	ApplicationStatus string `json:"applicationStatus"`
	AppliedOn         string `json:"appliedOn"`
	Description       string `json:"description"`
	UpdatedAt         string `json:"updatedAt"`
	UserId            string `json:"userId"`
}
