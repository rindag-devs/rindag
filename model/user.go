package model

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User represents a user.
type User struct {
	ID       uuid.UUID `gorm:"primary_key;type:uuid;default:uuid_generate_v4()"`
	Name     string    `gorm:"not null"`
	Password string    `gorm:"not null"`
	Email    string    `gorm:"not null"`
	TokenReq time.Time `gorm:"not null"`
}

// GetUser returns a user by Name / Email.
// The priority is given to Name.
func GetUser(db *gorm.DB, account string) (*User, error) {
	user := &User{}
	err := db.Where("name = ?", account).First(user).Error
	if err != nil {
		err = db.Where("email = ?", account).First(user).Error
	}
	return user, err
}

// GetUserById returns a user by ID.
func GetUserById(db *gorm.DB, id uuid.UUID) (*User, error) {
	user := &User{}
	err := db.Where("id = ?", id).First(user).Error
	return user, err
}

// Authenticate validates the password.
// Use bcrypt to compare the password.
func (user *User) Authenticate(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	return err == nil
}

// ExpireToken makes all the tokens of the user expired.
func (user *User) ExpireToken(db *gorm.DB) {
	user.TokenReq = time.Now()
	db.Save(&user)
}
