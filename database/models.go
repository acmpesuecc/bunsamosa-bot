package database

import (
	"gorm.io/gorm"
)

type DBManager struct {
	db *gorm.DB
}

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

type Maintainer struct {
	MaintainerID int `gorm:"primaryKey;not null;autoIncrement"`
	GithubHandle string
	Repos        []Repo
}

type Repo struct {
	RepoID  int `gorm:"primaryKey;not null;autoIncrement"`
	RepoURL string
	Issues  []Issue

	MaintainerID int
}

type Issue struct {
	IssueID  int `gorm:"primaryKey;not null;autoIncrement"`
	IssueURL string
	Status   bool
	Contributor Contributor

	RepoID      int
	// ContributorID int
}

type Contributor struct {
	ID           int `gorm:"primaryKey;not null;autoIncrement"`
	GithubHandle string

	IssueID int
}

// NOTE:
// select * from contributor join issue on contributor.assignedissueid = issue.issueid where contirbutor.githubhandle = anirudhsudhir
