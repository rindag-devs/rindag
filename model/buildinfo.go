package model

import (
	"time"

	"rindag/service/problem"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BuildInfo is the build information of the problem.
type BuildInfo struct {
	// Problem is the ID of the problem.
	Problem uuid.UUID `gorm:"type:uuid"`

	// Rev is the commit hash of the build.
	Rev []byte `gorm:"primary_key"`

	// BuildTime is the time of the build.
	BuildTime time.Time `gorm:"not null"`

	// Info is the build information.
	Info problem.BuildInfo `gorm:"not null"`
}

// UpdateBuildInfo creates a new build information.
func UpdateBuildInfo(
	db *gorm.DB, problem uuid.UUID, rev [20]byte, info problem.BuildInfo,
) (*BuildInfo, error) {
	buildInfo := &BuildInfo{
		Problem:   problem,
		Rev:       rev[:],
		BuildTime: time.Now(),
		Info:      info,
	}
	err := db.Save(buildInfo).Error
	return buildInfo, err
}
