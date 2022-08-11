package model

import (
	"time"

	"github.com/google/uuid"
)

// BuildInfo is the build information of the problem.
type BuildInfo struct {
	// Problem is the ID of the problem.
	Problem uuid.UUID `gorm:"primary_key,type:uuid"`

	// Rev is the commit hash of the build.
	Rev [20]byte `gorm:"not null"`

	// BuildTime is the time of the build.
	BuildTime time.Time `gorm:"not null"`
}
