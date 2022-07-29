package utils

import (
	"crypto/rand"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// TokenExpiration is the expiration time of the token.
const TokenExpiration = time.Hour * 24 * 7

var TokenSecret []byte

// UserClaims is the custom claims type for JWT.
type UserClaims struct {
	ID uuid.UUID `json:"id"`
	jwt.RegisteredClaims
}

// GenerateTOken generates a JWT token for the user.
func GenerateToken(id uuid.UUID) (string, error) {
	claims := UserClaims{
		ID: id,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "rindag",
			Subject:   "user",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExpiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
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
