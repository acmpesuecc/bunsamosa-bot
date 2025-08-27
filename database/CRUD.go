package database

import (
	"errors"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	gormlog "gorm.io/gorm/logger"
)

func (manager *DBManager) Init(connection_string string, logger *zerolog.Logger) error {
	manager.zeroLogger = logger
	dbInitLogger := manager.zeroLogger.With().Array("scope", zerolog.Arr().Str("DBMANAGER")).Logger()

	dbInitLogger.Info().Msg("Initializing Database...")
	db, err := gorm.Open(sqlite.Open(connection_string), &gorm.Config{
		Logger: gormlog.Default.LogMode(gormlog.Silent),
	})
	if err != nil {
		dbInitLogger.Error().Err(err).Msg("Could not initialize the database")
		return err
	} else {
		manager.db = db
		dbInitLogger.Info().Msg("Successfully Initialized Database")
	}

	dbInitLogger.Info().Msg("Beginning Model Automigration...")

	/* err = manager.db.AutoMigrate(&ContributorModel{})
	if err != nil {
		dbInitLogger.Error().Err(err).Msg("Could not AutoMigrate ContributorModel")
		return err
	}

	err = manager.db.AutoMigrate(&ContributorRecord{})
	if err != nil {
		dbInitLogger.Error().Err(err).Msg("Could not AutoMigrate ContributorRecord")
		return err
	}

	err = manager.db.AutoMigrate(&MaintainerModel{})
	if err != nil {
		dbInitLogger.Error().Err(err).Msg("Could not AutoMigrate MaintainerModel")
		return err
	} */

	err = manager.db.AutoMigrate(&Maintainer{})
	if err != nil {
		dbInitLogger.Error().Err(err).Msg("Could not AutoMigrate Maintainer")
	}

	err = manager.db.AutoMigrate(&Repo{})
	if err != nil {
		dbInitLogger.Error().Err(err).Msg("Could not AutoMigrate Repo")
	}

	err = manager.db.AutoMigrate(&MaintainerRepo{})
	if err != nil {
		dbInitLogger.Error().Err(err).Msg("Could not AutoMigrate MaintainerRepo")
	}

	err = manager.db.AutoMigrate(&Issue{})
	if err != nil {
		dbInitLogger.Error().Err(err).Msg("Could not AutoMigrate Issue")
	}

	err = manager.db.AutoMigrate(&Contributor{})
	if err != nil {
		dbInitLogger.Error().Err(err).Msg("Could not AutoMigrate Contributor")
	}

	err = manager.db.AutoMigrate(&ContributorIssue{})
	if err != nil {
		dbInitLogger.Error().Err(err).Msg("Could not AutoMigrate ContributorIssue")
	}

	err = manager.db.AutoMigrate(&BountyLogging{})
	if err != nil {
		dbInitLogger.Error().Err(err).Msg("Could not AutoMigrate BountyLogging")
	}

	err = manager.db.AutoMigrate(&ContributorBounty{})
	if err != nil {
		dbInitLogger.Error().Err(err).Msg("Could not AutoMigrate ContributorBounty")
	}

	dbInitLogger.Info().Msg("AutoMigration completed successfully")
	return nil
}

func (manager *DBManager) AssignBounty(
	maintainer string,
	contributor string,
	prHtmlUrl string,
	bountyPoints int,
) error {
	// TODO Handle for Re-assignment
	// Start a New Transaction to create this object

	// manager.sugaredLogger.Infof("Beginning Transaction to Assign Bounty",
	// 	zap.Strings("scope", []string{"DBMANAGER", "BOUNTY"}),
	// )

	manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("BOUNTY")).
		Msg("Beginning Transaction to Assign Bounty")
	// Create the dummy record for the contributor_model
	// contributor_model := ContributorModel{name: contributor}

	// Create the time-series record of this transaction
	// manager.sugaredLogger.Infof("Creating Contributor Record Model",
	// 	zap.Strings("scope", []string{"DBMANAGER", "BOUNTY"}),
	// )
	err := manager.db.Transaction(func(tx *gorm.DB) error {
		manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("BOUNTY")).
			Msg("Finding Contributor and Issue data")
		var contributorData Contributor
		result := tx.FirstOrCreate(&contributorData, Contributor{GithubHandle: contributor})
		if result.Error != nil {
			manager.zeroLogger.Error().Err(result.Error).Array("scope", zerolog.Arr().Str("DBMANAGER").Str("BOUNTY")).
				Msg("Could not find/create Contributor record")
			return result.Error
		}
		var issueData Issue
		result = tx.FirstOrCreate(&issueData, Issue{URL: prHtmlUrl})
		if result.Error != nil {
			manager.zeroLogger.Error().Err(result.Error).Array("scope", zerolog.Arr().Str("DBMANAGER").Str("BOUNTY")).
				Msg("Could not find/create Issue record")
			return result.Error
		}
		manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("BOUNTY")).
			Msg("Creating BountyLogging Model")
		bountyLog := BountyLogging{
			IssueID:        issueData.ID,
			ContributorID:  contributorData.ID,
			AssignedBounty: bountyPoints,
		}
		result = tx.Create(&bountyLog)
		if result.Error != nil {
			manager.zeroLogger.Error().Err(result.Error).Array("scope", zerolog.Arr().Str("DBMANAGER").Str("BOUNTY")).
				Msg("Could not create BountyLogging record")
			return result.Error
		}
		manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("BOUNTY")).
			Msg("Created BountyLogging record")

		manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("LEADERBOARD")).
			Msg("Beginning Recompute of ContributorBounty")

		// recalculate total bounties of contributors
		var contribBounties []ContributorBounty
		result = tx.Model(&BountyLogging{}).
			Select("contributor_id, SUM(assigned_bounty) as total_bounty, ? as updated_at", time.Now()).
			Group("contributor_id").
			Scan(&contribBounties)
		if result.Error != nil {
			manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("LEADERBOARD")).
				Err(result.Error).
				Msg("Could Not Recompute ContributorBounty")
			return result.Error
		}

		// updating total bounty (upsert)
		result = tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "contributor_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"total_bounty", "updated_at"}),
		}).Create(&contribBounties)
		if result.Error != nil {
			manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("LEADERBOARD")).
				Err(result.Error).
				Msg("Could Not update ContributorBounty")
			return result.Error
		}
		return nil
	})
	if err != nil {
		manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("LEADERBOARD")).
			Err(err).Msg("Transaction to assign bounty failed")
	}
	return err
}

func (manager *DBManager) GetAllRecords() ([]BountyLogging, error) {
	// Declare the array of all records
	var records []BountyLogging

	// Fetch from the database
	manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("RECORDS")).
		Msg("Fetching all records")
	fetch_result := manager.db.Find(&records)

	if fetch_result.Error != nil {
		manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("RECORDS")).
			Err(fetch_result.Error).
			Msg("Could not fetch all records")

		return nil, fetch_result.Error
	}

	return records, nil
}

func (manager *DBManager) GetUserRecords(contributor string) ([]BountyLogging, error) {
	// query := `select * from contributor_record_models
	//         where contributor_name like ?
	//         order by created_at desc;`

	// Declare the array of all records
	var records []BountyLogging

	// Fetch from the database
	manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("USER-SPECIFIC")).
		Msg("Fetching Records for user")

	fetch_result := manager.db.
		Joins("JOIN contributors ON contributors.id = bounty_loggings.contributor_id").
		Where("contributors.github_handle like ?", contributor).
		Preload("Issue").Order("bounty_loggings.created_at desc").Find(&records)
	// fetch_result := manager.db.Raw(query, contributor).Scan(&records)

	if fetch_result.Error != nil {
		manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("USER-SPECIFIC")).
			Err(fetch_result.Error).
			Msg("Could not fetch records")

		return nil, fetch_result.Error
	}
	return records, nil
}

func (manager *DBManager) GetLeaderboard() ([]LeaderboardEntry, error) {
	// leaderboard_query := `
	// SELECT contributor_name AS Name, sum(latest_points) AS Current_bounty from (
	// 	select
	// 		contributor_name, (SELECT points_allotted FROM contributor_record_models where t1.pullreq_url = pullreq_url order by created_at desc limit 1) as latest_points
	// 	from contributor_record_models as t1
	// 	GROUP by pullreq_url, contributor_name
	// ) GROUP BY contributor_name;
	// `
	// Declare the array of all records
	var records []LeaderboardEntry

	manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("LEADERBOARD")).
		Msg("Calculating leaderboard")

	fetch_result := manager.db.Model(&BountyLogging{}).
		Select("contributors.github_handle, SUM(bounty_loggings.assigned_bounty) as total_bounty").
		Joins("JOIN contributors ON contributors.id = bounty_loggings.contributor_id").
		Group("bounty_loggings.contributor_id, contributors.github_handle").
		Order("total_bounty DESC").Scan(&records)

	if fetch_result.Error != nil {
		manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("LEADERBOARD")).Err(fetch_result.Error).
			Msg("Failed to calculate leaderboard")
		return nil, fetch_result.Error
	}
	return records, nil
}

func (manager *DBManager) GetLeaderboardMat() ([]LeaderboardEntry, error) {
	// Declare the array of all records
	var records []ContributorBounty

	// Fetch from the database
	// manager.sugaredLogger.Infof("[DBMANAGER|MUX-LB] Fetching All Records")
	fetch_result := manager.db.Order("total_bounty DESC").Preload("Contributor").Find(&records)
	if fetch_result.Error != nil {
		manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("MUX-LB")).
			Err(fetch_result.Error).
			Msg("Could not fetch all records")
		return nil, fetch_result.Error
	}

	// convert to readable format
	var leaderboard []LeaderboardEntry
	for _, summary := range records {
		if summary.Contributor.ID != 0 {
			leaderboard = append(leaderboard, LeaderboardEntry{
				GithubHandle: summary.Contributor.GithubHandle,
				TotalBounty:  summary.TotalBounty,
			})
		}
	}
	return leaderboard, nil
}

func (manager *DBManager) CheckIsMaintainer(userName string) (bool, error) {
	var maintainer Maintainer

	manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("CHECK_MAINTAINER")).
		Str("username", userName).Msg("Checking valid maintainer")

	result := manager.db.Limit(1).First(&maintainer, "github_handle like ?", userName)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("CHECK_MAINTAINER")).
				Str("username", userName).Msg("No maintainer found")

			return false, nil
		}
		manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("CHECK_MAINTAINER")).
			Err(result.Error).Msg("Could not check for maintainer")

		return false, result.Error
	}

	manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("CHECK_MAINTAINER")).
		Str("username", userName).Msg("Maintainer found")

	return true, nil
}

// Check issue status and closure status before assigning issue
func (manager *DBManager) AssignIssue(issueURL string, contributorHandle string, repoURL string) (bool, error) {
	// Get issue_id from Issues table, create the issue record if it does not exist
	// Get the contributor_id from the Contributors table, create the contributor
	// record if it does not exist Check if the contributor has not been assigned
	// another issue Check if the issue has not been assigned to another
	// contributor If both the checks yield true, update the record in the
	// ContributorIssues table
	var repoData Repo
	var contributorData Contributor

	manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("ASSIGN")).
		Str("repo_url", repoURL).Msg("Obtaining id for repo from the Repos table")

	// Fetch the record with matching conditions or create a new record
	result := manager.db.FirstOrCreate(&repoData, &Repo{URL: repoURL})
	if result.Error != nil {
		manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("ASSIGN")).
			Err(result.Error).Str("repo_url", repoURL).Msg("Could not obtain repo from Repos table")

		return false, result.Error
	}

	manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("ASSIGN")).
		Str("issue_url", issueURL).Msg("Obtaining id for repo from the Issue table")

	// manager.sugaredLogger.Infof("Obtaining the id of issue %q from the Issues table\n", issueURL,
	// 	zap.Strings("scope", []string{"DBMANAGER", "ASSIGN"}),
	// )
	manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("ASSIGN")).
		Str("contributor_handle", contributorHandle).Msg("Obtaining contributor ID from table")

	result = manager.db.FirstOrCreate(&contributorData, &Contributor{GithubHandle: contributorHandle})
	if result.Error != nil {
		manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("ASSIGN")).
			Err(result.Error).Str("contributor_handle", contributorHandle).Msg("Could not obtain contributor from the Contributors table")

		return false, result.Error
	}

	err := manager.db.Transaction(func(tx *gorm.DB) error {
		var issueData Issue
		// Fetch the record with matching conditions or create a new record (lock row to prevent race)
		result = tx.Clauses(clause.Locking{Strength: "UPDATE"}).FirstOrCreate(&issueData, &Issue{URL: issueURL, RepoID: repoData.ID})
		if result.Error != nil {
			manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("ASSIGN")).
				Err(result.Error).Str("issue_url", issueURL).Msg("Could not obtain repo from Issues table")

			return result.Error
		}

		manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("ASSIGN")).
			Str("issue_url", issueData.URL).Msg("Checking if issue has been assigned")

		if issueData.Status {
			manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("ASSIGN")).
				Err(result.Error).Str("issue_url", issueData.URL).Msg("Issue has already been assigned to someone else")

			return errors.New("issue is already assigned to someone")
		}

		manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("ASSIGN")).
			Str("contributor_handle", contributorHandle).Msg("Storing assignment of issue for contributor if contributor has not already been assigned an issue")

		// as contributor_issue.contributor_id is a unique index, it will fail if already populated
		assignErr := tx.Model(&issueData).Association("Contributors").Append(&contributorData)

		if assignErr != nil {
			if strings.Contains(assignErr.Error(), "UNIQUE constraint failed") {
				manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("ASSIGN")).
					Err(assignErr).Str("contributor_handle", contributorHandle).Int("contributor_id", contributorData.ID).Int("issue_data", issueData.ID).
					Msg("Contributor has already been assigned with an issue")
				return errors.New("contributor is already assigned to another issue")
			}
			return assignErr
		}

		issueData.Status = true
		result = tx.Save(&issueData)
		if result.Error != nil {
			manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("ASSIGN").Str("TRANSACTION")).
				Err(result.Error).Int("issue_id", issueData.ID).Int("contributor_id", contributorData.ID).Str("contributor_handle", contributorHandle).
				Msg("Could not store assignment of issue for contributor")
			return result.Error
		}

		return nil
	})
	if err != nil {
		manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("ASSIGN").Str("TRANSACTION")).
			Err(result.Error).Str("issue_url", issueURL).Int("contributor_id", contributorData.ID).Str("contributor_handle", contributorHandle).
			Msg("Could not store assignment of issue for contributor")
		return false, err
	}

	return true, nil
}

func (manager *DBManager) DeassignIssue(issueURL string) (bool, error) {
	err := manager.db.Transaction(func(tx *gorm.DB) error {
		var issueData Issue

		manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("DEASSIGN")).
			Str("issue_url", issueURL).Msg("Obtaining the id of issue from Issue Table")

		// Fetch the issue record from the Issues table
		result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("url = ?", issueURL).First(&issueData)
		if result.Error != nil {
			// If the issue is not found, log and return false without an error (check for this outside transaction)
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("DEASSIGN")).
					Str("issue_url", issueURL).Msg("Issue not found in Issue Table")
				return errors.New("custom: issue not found")
			}
			manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("DEASSIGN")).
				Err(result.Error).Str("issue_url", issueURL).
				Msg("Could not obtain issue from Issue table")
			return result.Error
		}

		manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("DEASSIGN")).
			Int("issue_id", issueData.ID).Msg("Checking if issue has been assigned to any contributor")

		if !issueData.Status {
			// If no contributor is assigned to the issue, log and return false without an error
			manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("DEASSIGN")).
				Str("issue_url", issueURL).Msg("No contributor assigned to the issue")
			// we should ideally just return nil, but if return an error, we can check outside transaction
			// and rollback of transaction shouldn't matter here as we've simply queried
			return errors.New("custom: issue already unassigned")
		}

		manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("DEASSIGN")).
			Int("issue_id", issueData.ID).Int("contributor_id", issueData.Contributors[0].ID).
			Msg("Removing assignment of issue with contributor")

		// Start a new transaction
		// err := manager.db.Transaction(func(tx *gorm.DB) error {

		clearErr := tx.Model(&issueData).Association("Contributors").Clear()
		if clearErr != nil {
			manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("DEASSIGN")).
				Err(result.Error).Int("issue_id", issueData.ID).Int("contributor_id", issueData.Contributors[0].ID).
				Msg("Could not remove assignment of issue for contributor")
			return result.Error
		}

		// Update the status of the issue, change to bool later
		issueData.Status = false
		result = tx.Save(&issueData)
		if result.Error != nil {
			manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("DEASSIGN")).
				Err(result.Error).Int("issue_id", issueData.ID).Msg("Could not update status of issue")
			return result.Error
		}
		return nil
	})
	if err != nil {
		if strings.Contains(err.Error(), "custom:") {
			return false, nil
		}
		manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("DEASSIGN").Str("TRANSACTION")).
			Err(err).Str("issue_url", issueURL).Msg("Failed to deassign issue in transaction")
		return false, err
	}

	return true, nil
}

func (manager *DBManager) WithdrawIssue(issueURL string, contributorHandle string) (bool, error) {
	// While withdrawing an issue we need to make sure to check
	// within the db that the contributor requesting for the
	// deassign is assigned to the issue with the same url
	//
	// we need to update the issues table and set its status
	// to false, indicating the issue is now available to be
	// contributed to by another person

	err := manager.db.Transaction(func(tx *gorm.DB) error {
		var issueData Issue
		var contributorData Contributor

		manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("WITHDRAW")).
			Msg("Obtaining the id of the issue from the Issue table")

		result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("url = ?", issueURL).First(&issueData)
		if result.Error != nil {
			manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("WITHDRAW")).
				Str("issue_url", issueURL).Msg("Could not obtain the id of the issue from URL")

			return result.Error
		}

		manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("WITHDRAW")).
			Msg("Obtaining the id of the contributor from handle")

		result = tx.Where("github_handle = ?", contributorHandle).First(&contributorData)
		if result.Error != nil {
			manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("WITHDRAW")).
				Str("contributor_handle", contributorHandle).Msg("Could not obtain the id of the contributor from handle")

			return result.Error
		}

		manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("WITHDRAW")).
			Str("contributor_handle", contributorHandle).Str("issue_url", issueURL).
			Msg("Checking if the contributor is assigned to the request issue")

		var contributorIssueData ContributorIssue
		result = tx.Where("issue_id = ? and contributor_id = ?", issueData.ID, contributorData.ID).First(&contributorIssueData)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("WITHDRAW")).
					Str("contributor_handle", contributorHandle).Int("contributor_id", contributorData.ID).Int("issue_id", issueData.ID).
					Msg("Contributor requested for deassign on an issue they're not assigned to")
				return errors.New("custom: contributor not assigned to this issue")
			}
			manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("WITHDRAW")).
				Str("contributor_handle", contributorHandle).Int("contributor_id", contributorData.ID).
				Msg("Could not query contributor from ContributorIssues Table")
			return result.Error
		}

		deleteErr := tx.Model(&issueData).Association("Contributors").Delete(&contributorData)
		if deleteErr != nil {
			manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("WITHDRAW").Str("TRANSACTION")).
				Err(deleteErr).Str("contributor_handle", contributorHandle).Int("issue_id", issueData.ID).
				Msg("Failed to delete contributor<->issue link")
			return deleteErr
		}

		issueData.Status = false
		transaction_result := tx.Save(&issueData)
		if transaction_result.Error != nil {
			manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("WITHDRAW").Str("TRANSACTION")).
				Err(transaction_result.Error).Str("issue_url", issueURL).Int("issue_id", issueData.ID).
				Msg("Failed to update issue to status false (available)")
			return transaction_result.Error
		}

		return nil
	})
	if err != nil {
		if strings.Contains(err.Error(), "custom") {
			return false, nil
		}
		manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("WITHDRAW").Str("TRANSACTION")).
			Err(err).Str("issue_url", issueURL).Str("contributor_handle", contributorHandle).
			Msg("Could not perform transaction to withdraw contributor from issue")
		return false, err
	}

	return true, nil
}

// CRUD op to check assign status for an identified contrib
func (manager *DBManager) CheckUserAssigned(contributorHandle string) (bool, string, error) {
	var contributorData Contributor

	manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("ASSIGN")).Str("contributor_handle", contributorHandle).Msg("Obtaining the id of the contributor from the Contributors table")
	result := manager.db.Preload("Issues").Where("github_handle = ?", contributorHandle).First(&contributorData)
	if result.Error != nil {
		// If the error is a missing record, continue to assign the issue and add a record to the table
		// Else, return the error
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("CHECK_USER_ASSIGNED")).Str("contributor_handle", contributorHandle).Msg("Contributor does not exist in the database")
			return false, "", nil
		}

		manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("CHECK_USER_ASSIGNED")).Err(result.Error).Str("contributor_handle", contributorHandle).Msg("Could not obtain contributor from the Contributors table")

		return false, "", result.Error
	}

	manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("CHECK_USER_ASSIGNED")).Str("contributor_handle", contributorHandle).Msg("Checking if contributor is already assigned to an issue")
	// currently we are only allowing a single contributor to be assigned to a single issue
	if len(contributorData.Issues) == 1 {
		assignedIssue := contributorData.Issues[0]
		manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("CHECK_USER_ASSIGNED")).Err(result.Error).Str("contributor_handle", contributorHandle).Int("contributor_issue_id", assignedIssue.ID).Msg("Contributor is assigned to another issue")
		return true, assignedIssue.URL, nil
	}
	manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("CHECK_USER_ASSIGNED")).Str("contributor_handle", contributorHandle).Msg("Contributor is not assigned to any issue")
	return false, "", nil
}
