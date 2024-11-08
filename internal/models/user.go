package models

import "time"

type User struct {
	ID           int64     `json:"-" db:"id"`
	Login        string    `json:"login" db:"login"`
	PasswordHash string    `json:"-" db:"password_hash"`
	CreatedAt    time.Time `json:"-" db:"created_at"`
}

type Credentials struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}
