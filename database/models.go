package database

import (
	"time"

	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

type DBManager struct {
	db         *gorm.DB
	zeroLogger *zerolog.Logger
}

// type ContributorModel struct {
// 	gorm.Model

// 	Name           string
// 	Current_bounty int `gorm:"default:0"`
// }

// type MaintainerModel struct {
// 	Username string `gorm:"primaryKey"`
// }

// type ContributorRecordModel struct {
// 	gorm.Model

// 	Contributor_name string
// 	Maintainer_name  string
// 	Pullreq_url      string
// 	Points_allotted  int
// }

type Maintainer struct {
	ID           int    `gorm:"primaryKey;not null;autoIncrement"`
	GithubHandle string `gorm:"uniqueIndex"`

	Repos []Repo `gorm:"many2many:maintainer_repos;"`
}

type Repo struct {
	ID  int    `gorm:"primaryKey;not null;autoIncrement"`
	URL string `gorm:"uniqueIndex"`

	Issues      []Issue      `gorm:"foreignKey:RepoID"`
	Maintainers []Maintainer `gorm:"many2many:maintainer_repos;"`
}

type MaintainerRepo struct {
	// Foreign keys part of MaintainerRepo
	MaintainerID int `gorm:"primaryKey;index"`
	RepoID       int `gorm:"primaryKey;index"`
}

type Issue struct {
	ID     int    `gorm:"primaryKey;not null;autoIncrement"`
	URL    string `gorm:"uniqueIndex"`
	Status bool   `gorm:"default:0"`
	Closed bool   `gorm:"default:0"`

	// Foreign keys part of Issue
	RepoID int
	Repo   Repo `gorm:"foreignKey:RepoID"`

	Contributors []Contributor   `gorm:"many2many:contributor_issues;"`
	BountyLogs   []BountyLogging `gorm:"foreignKey:IssueID"`
}

type Contributor struct {
	ID           int    `gorm:"primaryKey;not null;autoIncrement"`
	GithubHandle string `gorm:"default:null"`

	Issues     []Issue         `gorm:"many2many:contributor_issues;"`
	BountyLogs []BountyLogging `gorm:"foreignKey:ContributorID"`
}

// join table for contributor and issues
type ContributorIssue struct {

	// Foreign keys part of ContributorIssue
	ContributorID int `gorm:"primaryKey;uniqueIndex:idx_active_assignee"`
	IssueID       int `gorm:"primaryKey;index"`
}

// Append-only
type BountyLogging struct {
	ID             int `gorm:"primaryKey;not null;autoIncrement;<-:create"`
	AssignedBounty int `gorm:"default:0;<-:create;index:idx_bounty,priority:3"`
	CreatedAt      time.Time

	// Foreign Key's part of BountyLogging
	ContributorID int         `gorm:"<-:create;index:idx_bounty,priority:1"`
	IssueID       int         `gorm:"<-:create;index:idx_bounty,priority:2"`
	Contributor   Contributor `gorm:"foreignKey:ContributorID"`
	Issue         Issue       `gorm:"foreignKey:IssueID"`
}

// Materialised table for countributor bounty
type ContributorBounty struct {
	ID            int         `gorm:"primaryKey"`
	ContributorID int         `gorm:"uniqueIndex"`
	Contributor   Contributor `gorm:"foreignKey:ContributorID"`
	TotalBounty   int         `gorm:"default:0"`
	UpdatedAt     time.Time
}

// used for displaying the leaderboard as ContributorModel is deprecated
type LeaderboardEntry struct {
	GithubHandle string
	TotalBounty  int
}
