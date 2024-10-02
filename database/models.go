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


