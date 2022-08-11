package model

import (
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// Problem represents a problem in database.
type Problem struct {
	ID           uuid.UUID      `gorm:"primary_key;type:uuid;default:uuid_generate_v4()" json:"id"`
	Name         string         `gorm:"not null" json:"name"`
	Tags         pq.StringArray `gorm:"not null;type:text[]" json:"tags"`
	LastBuildRev *string        `json:"last_build_rev"`
}

// GetProblemIDsList returns a list of IDs of all problem.
func GetProblemIDsList(db *gorm.DB) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := db.Find(&ids).Error
	return ids, err
}

// GetProblemByID returns a problem by ID.
func GetProblemByID(db *gorm.DB, id uuid.UUID) (*Problem, error) {
	var problem Problem
	err := db.Where("id = ?", id).First(&problem).Error
	return &problem, err
}

// SearchProblemByName returns a list of problems whose name contains the given name.
func SearchProblemByName(db *gorm.DB, name string) ([]Problem, error) {
	var problems []Problem
	err := db.Where("name LIKE ?", "%"+name+"%").Find(&problems).Error
	return problems, err
}

// CreateProblem creates a new problem.
//
// Usually the BuildResult is empty.
func CreateProblem(db *gorm.DB, name string, tags []string) (*Problem, error) {
	problem := &Problem{
		Name: name,
		Tags: tags,
	}
	err := db.Create(problem).Error
	return problem, err
}

// ListProblems returns a list of problems.
func ListProblems(db *gorm.DB) ([]Problem, error) {
	var problems []Problem
	err := db.Model(&Problem{}).Find(&problems).Error
	return problems, err
}
