package utils

import (
	"crypto/rand"
	"errors"
	"time"

	"rindag/model"
	"rindag/service/db"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// TokenExpiration is the expiration time of the token.
const TokenExpiration = time.Hour * 24 * 7

var (
	TokenSecret         []byte
	ErrTokenInvalidUser = errors.New("token has invalid user")
	ErrTokenNotValidYet = errors.New("token is not valid yet")
	ErrTokenExpired     = errors.New("token is expired")
)

// UserClaims is the custom claims type for JWT.
type UserClaims struct {
	ID        uuid.UUID `json:"id"`
	IssuedAt  time.Time `json:"iat"`
	ExpiresAt time.Time `json:"exp"`
}

// Valid returns an error if the token is invalid.
func (c UserClaims) Valid() error {
	now := time.Now()

	if c.IssuedAt.After(now) {
		return ErrTokenNotValidYet
	}

	if c.ExpiresAt.Before(now) {
		return ErrTokenExpired
	}

	user, err := model.GetUserById(db.PDB, c.ID)
	if err != nil {
		return ErrTokenInvalidUser
	}

	if c.IssuedAt.Before(user.TokenReq) {
		// Token is issued before the the last logout.
		// When user logout, the token is invalid.
		return ErrTokenExpired
	}

	return nil
}

// GenerateTOken generates a JWT token for the user.
func GenerateToken(id uuid.UUID) (string, error) {
	now := time.Now()
	claims := UserClaims{
		ID:        id,
		IssuedAt:  now,
		ExpiresAt: now.Add(TokenExpiration),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(TokenSecret)
}

// ParseToken parses a JWT token.
func ParseToken(token string) (*UserClaims, error) {
	claims := &UserClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return TokenSecret, nil
	})
	if err != nil {
		return nil, err
	}
	return claims, nil
}

func init() {
	rndUUID, err := uuid.NewRandomFromReader(rand.Reader)
	if err != nil {
		log.WithError(err).Fatal("Error creating UUID")
	}
	TokenSecret, err = rndUUID.MarshalBinary()
	if err != nil {
		log.WithError(err).Fatal("Error creating UUID")
	}
}
