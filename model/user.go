package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User represents a user.
type User struct {
	ID       uuid.UUID `gorm:"primary_key;type:uuid;default:uuid_generate_v4()"`
	Name     string    `gorm:"not null"`
	Password string    `gorm:"not null"`
	Email    string    `gorm:"not null"`
}

// GetUser returns a user by Name / Email.
// The priority is given to Name.
func GetUser(db *gorm.DB, pat string) (*User, error) {
	user := &User{}
	err := db.Where("name = ?", pat).First(user).Error
	if err != nil {
		err = db.Where("email = ?", pat).First(user).Error
	}
	return user, err
}
