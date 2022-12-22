// Package internal provides
package internal

// User .
type User struct {
	ID   int64
	Name string
}

// TableName .
func (u *User) TableName() string {
	return "users"
}
