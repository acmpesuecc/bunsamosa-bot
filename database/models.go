package database

import "gorm.io/gorm"

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
	ID           int `gorm:"primaryKey;not null;autoIncrement"`
	GithubHandle string

	MaintainerRepos []MaintainerRepo
}

type Repo struct {
	ID  int `gorm:"primaryKey;not null;autoIncrement"`
	URL string

	Issues          []Issue
	MaintainerRepos []MaintainerRepo
}

type MaintainerRepo struct {
	Id int `gorm:"primaryKey;not null;autoIncrememnt"`

	// Foreign keys part of MaintainerRepo
	MaintainerID int
	RepoID       int
}

type Issue struct {
	ID     int    `gorm:"primaryKey;not null;autoIncrement"`
	URL    string `gorm:"default:null"`
	Status bool `gorm:"default:0"`
	Closed bool   `gorm:"default:0"`

	// Foreign keys part of Issue
	RepoID int

	ContributorIssues []ContributorIssue
	BountyLogs        []BountyLogging
}

type Contributor struct {
	ID           int    `gorm:"primaryKey;not null;autoIncrement"`
	GithubHandle string `gorm:"default:null"`

	ContributorIssue ContributorIssue
	BountyLogs       []BountyLogging
}

type ContributorIssue struct {
	ID int `gorm:"primaryKey;not null;autoIncrememnt"`

	// Foreign keys part of ContributorIssue
	ContributorID int
	IssueID       int
}

// Append-only
type BountyLogging struct {
	ID             int `gorm:"primaryKey;not null;autoIncrememnt;<-:create"`
	AssignedBounty int `gorm:"default:0;<-:create"`

	// Foreign Key's part of BountyLogging
	ContributorID int `gorm:"<-:create"`
	IssueID       int `gorm:"<-:create"`
}
