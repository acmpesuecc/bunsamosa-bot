package database

import (
	"gorm.io/gorm"

)

type ContributorModel struct {
	gorm.Model

	Name           string
	Current_bounty int `gorm:"default:0"`
}

type MaintainerModel struct {
	Username string `gorm:"primaryKey"`
}

type ContributorRecordModel struct {
	gorm.Model

	Contributor_name string
	Maintainer_name  string
	Pullreq_url      string
	Points_allotted  int
}

// TODO Implement method to connect GORM based on connection
// String
// Return GORM instance to store on main struct

// Manager struct
type DBManager struct {
	db *gorm.DB
}

type Maintainer struct {
	ID     int    `gorm:"primaryKey"`
	Handle string
	Repos  []Repo `gorm:"foreignKey:MaintainerID"`
}

type Repo struct {
	RepoID       int    `gorm:"primaryKey"`
	RepoURL      string
	MaintainerID int
	Maintainer   Maintainer
	Issues       []Issue `gorm:"foreignKey:RepoID"`
}

type Issue struct {
	IssueID int `gorm:"primaryKey"`
	IssueURL string
	RepoID   int
	Repo     Repo
	Status   bool
	Contributors []Contributor `gorm:"foreignKey:AssignedIssueID"`
}

type Contributor struct {
	ID               int `gorm:"primaryKey"`
	Handle           string
	AssignedIssueID  int
	AssignedIssue    Issue
}

