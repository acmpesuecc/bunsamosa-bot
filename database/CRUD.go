package database

import (
	"errors"

	"github.com/rs/zerolog"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func (manager *DBManager) Init(connection_string string, logger *zerolog.Logger) error {
	manager.zeroLogger = logger
	dbInitLogger := manager.zeroLogger.With().Array("scope", zerolog.Arr().Str("DBMANAGER")).Logger()

	dbInitLogger.Info().Msg("Initializing Database...")
	db, err := gorm.Open(sqlite.Open(connection_string), &gorm.Config{})
	if err != nil {
		dbInitLogger.Error().Err(err).Msg("Could not initialize the database")
		return err
	} else {
		manager.db = db
		dbInitLogger.Info().Msg("Successfully Initialized Database")
	}

	dbInitLogger.Info().Msg("Beginning Model Automigration...")

	err = manager.db.AutoMigrate(&ContributorModel{})
	if err != nil {
		dbInitLogger.Error().Err(err).Msg("Could not AutoMigrate ContributorModel")
		return err
	}

	err = manager.db.AutoMigrate(&ContributorRecordModel{})
	if err != nil {
		dbInitLogger.Error().Err(err).Msg("Could not AutoMigrate ContributorRecordModel")
		return err
	}

	err = manager.db.AutoMigrate(&MaintainerModel{})
	if err != nil {
		dbInitLogger.Error().Err(err).Msg("Could not AutoMigrate MaintainerModel")
		return err
	}

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

	dbInitLogger.Info().Msg("AutoMigration completed successfully")
	return nil
}

func (manager *DBManager) AssignBounty(
	maintainer string,
	contributor string,
	pr_html_url string,
	bounty_points int,
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
	manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("BOUNTY")).
		Msg("Creating Contributor Record Model")

	crm := ContributorRecordModel{
		Maintainer_name:  maintainer,
		Contributor_name: contributor,
		Pullreq_url:      pr_html_url,
		Points_allotted:  bounty_points,
	}

	// Create the user struct
	// contributor_temp_representation := ContributorModel{
	// 	Name:           contributor,
	// 	Current_bounty: bounty_points,
	// }

	manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("BOUNTY")).
		Msg("Created Contributor Record Model")

	manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("BOUNTY")).
		Any("crm", crm).
		Msg("Beginning Transaction")

	// TODO: Transaction error not handled
	manager.db.Transaction(func(tx *gorm.DB) error {
		// Create the time-series record
		result := tx.Create(&crm)
		if result.Error != nil {

			// Edge Case - User record already exists in time-series data
			// In that case, update that

			manager.zeroLogger.Error().
				Array("scope", zerolog.Arr().Str("DBMANAGER").Str("BOUNTY")).
				Err(result.Error).
				Msg("Could Not Create ContributorRecordModel")

			return result.Error
		}

		// default case - assume the user does not exist

		/*
			// Test if the user exists by attempting to create the user as
			// a new record
			user_create_result := tx.Create(&contributor_temp_representation)

			if user_create_result.Error != nil {
				// Check for the case where the user already exists

				// if that's the case, update the bounty with the new points

				// Else, report the error -> We found somethin unexpected

			} else {
				// Set the Bounty values
				// No Error, you can use this newly created user
				return nil
			}
		*/

		manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("LEADERBOARD")).
			Msg("Beginning Recompute of ContributorModel")

		// Recompute ContributorModel Table
		lb_query := `DELETE FROM contributor_models;INSERT INTO contributor_models (Name, Current_bounty)
SELECT contributor_name AS Name, sum(latest_points) AS Current_bounty from (
   select
       contributor_name, (SELECT points_allotted FROM contributor_record_models where t1.pullreq_url = pullreq_url order by created_at desc limit 1) as latest_points
   from contributor_record_models as t1
   GROUP by pullreq_url, contributor_name
) GROUP BY contributor_name;`

		result = tx.Exec(lb_query)
		if result.Error != nil {
			manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("LEADERBOARD")).
				Err(result.Error).
				Msg("Could Not Recompute ContributorModel")
			return result.Error
		}
		// commit the transaction
		return nil
	})

	return nil
}

func (manager *DBManager) GetAllRecords() ([]ContributorRecordModel, error) {
	// Declare the array of all records
	var records []ContributorRecordModel

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

func (manager *DBManager) GetUserRecords(contributor string) ([]ContributorRecordModel, error) {
	query := `select * from contributor_record_models
         where contributor_name like ?
         order by created_at desc;`

	// Declare the array of all records
	var records []ContributorRecordModel

	// Fetch from the database
	manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("USER-SPECIFIC")).
		Msg("Fetching Records for user")

	fetch_result := manager.db.Raw(query, contributor).Scan(&records)

	if fetch_result.Error != nil {
		manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("USER-SPECIFIC")).
			Err(fetch_result.Error).
			Msg("Could not fetch records")

		return nil, fetch_result.Error
	}
	return records, nil
}

func (manager *DBManager) GetLeaderboard() ([]ContributorModel, error) {
	leaderboard_query := `
	SELECT contributor_name AS Name, sum(latest_points) AS Current_bounty from (
		select
			contributor_name, (SELECT points_allotted FROM contributor_record_models where t1.pullreq_url = pullreq_url order by created_at desc limit 1) as latest_points
		from contributor_record_models as t1
		GROUP by pullreq_url, contributor_name
	) GROUP BY contributor_name;
	`

	// Declare the array of all records
	var records []ContributorModel

	// Fetch from the database
	manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("LEADERBOARD")).
		Msg("Fetching All Records")

	fetch_result := manager.db.Raw(leaderboard_query).Scan(&records)

	if fetch_result.Error != nil {
		manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("LEADERBOARD")).
			Err(fetch_result.Error).
			Msg("Could not fetch all records")

		return nil, fetch_result.Error
	}
	return records, nil
}

func (manager *DBManager) GetLeaderboardMat() ([]ContributorModel, error) {
	// Declare the array of all records
	var records []ContributorModel

	// Fetch from the database
	// manager.sugaredLogger.Infof("[DBMANAGER|MUX-LB] Fetching All Records")
	fetch_result := manager.db.Find(&records)
	if fetch_result.Error != nil {
		manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("MUX-LB")).
			Err(fetch_result.Error).
			Msg("Could not fetch all records")
		return nil, fetch_result.Error
	}
	return records, nil
}

func (manager *DBManager) CheckIsMaintainer(user_name string) (bool, error) {
	var maintainer MaintainerModel

	manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("CHECK_MAINTAINER")).
		Str("username", user_name).Msg("Checking valid maintainer")

	result := manager.db.Limit(1).First(&maintainer, "username like ?", user_name)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("CHECK_MAINTAINER")).
				Str("username", user_name).Msg("No maintainer found")

			return false, nil
		}
		manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("CHECK_MAINTAINER")).
			Err(result.Error).Msg("Could not check for maintainer")

		return false, result.Error
	}

	manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("CHECK_MAINTAINER")).
		Str("username", user_name).Msg("Maintainer found")

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
	var issueData Issue
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
	// Fetch the record with matching conditions or create a new record
	result = manager.db.FirstOrCreate(&issueData, &Issue{URL: issueURL, RepoID: repoData.ID})
	if result.Error != nil {
		manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("ASSIGN")).
			Err(result.Error).Str("issue_url", issueURL).Msg("Could not obtain repo from Issues table")

		return false, result.Error
	}

	manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("ASSIGN")).
		Str("issue_url", issueData.URL).Msg("Checking if issue has been assigned")

	if issueData.Status {
		manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("ASSIGN")).
			Err(result.Error).Str("issue_url", issueData.URL).Msg("Issue has already been assigned to someone else")

		return false, result.Error
	}

	manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("ASSIGN")).
		Str("contributor_handle", contributorHandle).Msg("Obtaining contributor ID from table")

	result = manager.db.FirstOrCreate(&contributorData, &Contributor{GithubHandle: contributorHandle})
	if result.Error != nil {
		manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("ASSIGN")).
			Err(result.Error).Str("contributor_handle", contributorHandle).Msg("Could not obtain contributor from the Contributors table")

		return false, result.Error
	}

	manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("ASSIGN")).
		Str("contributor_handle", contributorHandle).Msg("Checking if contributor has been assigned an issue")

	var contributorIssue ContributorIssue
	result = manager.db.Find(&contributorIssue, "contributor_id = ?", contributorData.ID)

	if result.Error != nil {
		// If the error is a missing record, continue to assign the issue and add a record to the table
		// Else, return the error
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("ASSIGN")).
				Str("contributor_handle", contributorHandle).Int("contributor_id", contributorData.ID).
				Msg("Could not query contributor from ContributorIssue table")
			return false, result.Error
		}
	} else {
		// If contributor is assigned another issue (IssueID != 0), return false without an error
		if contributorIssue.IssueID != 0 {
			manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("ASSIGN")).
				Err(result.Error).Str("contributor_handle", contributorHandle).Int("contributor_id", contributorData.ID).Int("issue_data", issueData.ID).
				Msg("Contributor has already been assied with an issue")
			return false, nil
		}
	}

	manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("ASSIGN")).
		Str("contributor_handle", contributorHandle).Int("contributor_id", contributorData.ID).Int("issue_id", issueData.ID).
		Msg("Storing assignment of issue for contributor")

	result = manager.db.FirstOrCreate(&contributorIssue, ContributorIssue{IssueID: issueData.ID})
	if result.Error != nil {
		manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("ASSIGN")).
			Err(result.Error).Int("issue_id", issueData.ID).
			Msg("Could not obtain issue from ContributorIssue table")

		return false, result.Error
	}

	err := manager.db.Transaction(func(tx *gorm.DB) error {
		contributorIssue.ContributorID = contributorData.ID
		result = tx.Save(&contributorIssue)
		if result.Error != nil {
			manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("ASSIGN").Str("TRANSACTION")).
				Err(result.Error).Int("issue_id", issueData.ID).Int("contributor_id", contributorData.ID).Str("contributor_handle", contributorHandle).
				Msg("Could not obtain issue from ContributorIssue table")
			return result.Error
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
			Err(result.Error).Int("issue_id", issueData.ID).Int("contributor_id", contributorData.ID).Str("contributor_handle", contributorHandle).
			Msg("Could not store assignment of issue for contributor")
		return false, err
	}

	return true, nil
}

func (manager *DBManager) DeassignIssue(issueURL string) (bool, error) {
	var issueData Issue
	var contributorIssue ContributorIssue

	manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("DEASSIGN")).
		Str("issue_url", issueURL).Msg("Obtaining the id of issue from Issue Table")

	// Fetch the issue record from the Issues table
	result := manager.db.First(&issueData, "url LIKE ?", issueURL)
	if result.Error != nil {
		// If the issue is not found, log and return false without an error, or modify if u want error
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("DEASSIGN")).
				Str("issue_url", issueURL).Msg("Issue not found in Issue Table")
			return false, nil
		}
		manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("DEASSIGN")).
			Err(result.Error).Str("issue_url", issueURL).
			Msg("Could not obtain issue from Issue table")
		return false, result.Error
	}

	manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("DEASSIGN")).
		Int("issue_id", issueData.ID).Msg("Checking if issue has been assigned to any contributor")

	// Fetch the contributor issue record from the ContributorIssues table
	result = manager.db.Find(&contributorIssue, "issue_id = ?", issueData.ID)
	if result.Error != nil {
		// If no contributor is assigned to the issue, log and return false without an error
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("DEASSIGN")).
				Str("issue_url", issueURL).Msg("No contributor assigned to the issue")
			return false, nil
		}
		// manager.sugaredLogger.Errorf("Could not query issue %q with IssueId %d from the ContributorIssues table\n", issueURL, issueData.ID,
		// 	zap.Strings("scope", []string{"DBMANAGER", "DEASSIGN"}),
		// )
		manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("DEASSIGN")).
			Err(result.Error).Str("issue_url", issueURL).Int("issue_id", issueData.ID).
			Msg("Could not query ContributorIssue Table")
		return false, result.Error
	}

	manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("DEASSIGN")).
		Int("issue_id", issueData.ID).Int("contributor_id", contributorIssue.ContributorID).
		Msg("Removing assignment of issue with contributor")

	// Start a new transaction
	err := manager.db.Transaction(func(tx *gorm.DB) error {
		// Set the contributor ID to 0 in the ContributorIssues table
		contributorIssue.ContributorID = 0
		result = tx.Save(&contributorIssue)
		if result.Error != nil {
			manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("DEASSIGN")).
				Err(result.Error).Int("issue_id", issueData.ID).Int("contributor_id", contributorIssue.ContributorID).
				Msg("Could not remove assignment of issue for contributor")
			return result.Error
		}

		// Update the status of the issue, change to bool later
		issueData.Status = false
		result = tx.Save(&issueData)
		if result.Error != nil {
			manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("DEASSIGN")).
				Err(result.Error).Int("issue_id", issueData.ID).Msg("Could not udpate status of issue")
			return result.Error
		}

		return nil
	})
	if err != nil {
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

	var issueData Issue
	var contributorData Contributor

	manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("WITHDRAW")).
		Msg("Obtaining the id of the issue from the Issue table")

	result := manager.db.First(&issueData, &Issue{URL: issueURL})
	if result.Error != nil {
		manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("WITHDRAW")).
			Str("issue_url", issueURL).Msg("Could not obtain the id of the issue from URL")

		return false, result.Error
	}

	manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("WITHDRAW")).
		Msg("Obtaining the id of the contributor from handle")

	result = manager.db.First(&contributorData, &Contributor{GithubHandle: contributorHandle})
	if result.Error != nil {
		manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("WITHDRAW")).
			Str("contributor_handle", contributorHandle).Msg("Could not obtain the id of the contributor from handle")

		return false, result.Error
	}

	manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("WITHDRAW")).
		Str("contributor_handle", contributorHandle).Str("issue_url", issueURL).
		Msg("Checking if the contributor is assigned to the request issue")

	var contributorIssueData ContributorIssue
	result = manager.db.First(&contributorIssueData, "contributor_id = ?", contributorData.ID)
	if result.Error != nil {
		manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("WITHDRAW")).
			Str("contributor_handle", contributorHandle).Int("contributor_id", contributorData.ID).
			Msg("Could not query contributor from ContributorIssues Table")
		return false, result.Error
	} else {
		// If contributor is assigned another issue, and there is a mismatch between
		// the issue withdraw has been requested from and what they have been assigned to
		if contributorIssueData.IssueID != issueData.ID {
			manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("WITHDRAW")).
				Str("contributor_handle", contributorHandle).Int("contributor_id", contributorData.ID).Int("issue_id", issueData.ID).
				Msg("Contributor requested for deassign on a different issue")
			return false, nil
		}
	}

	err := manager.db.Transaction(func(tx *gorm.DB) error {
		contributorIssueData.IssueID = 0
		transaction_result := tx.Save(&contributorIssueData)
		if transaction_result.Error != nil {
			manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("WITHDRAW").Str("TRANSACTION")).
				Err(transaction_result.Error).Str("contributor_handle", contributorHandle).Int("issue_id", issueData.ID).
				Msg("Failed to update contributorIssueData.IssueID")
			return transaction_result.Error
		}

		issueData.Status = false
		transaction_result = tx.Save(&issueData)
		if transaction_result.Error != nil {
			manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("WITHDRAW").Str("TRANSACTION")).
				Err(transaction_result.Error).Str("issue_url", issueURL).Int("issue_id", issueData.ID).
				Msg("Failed to update issue to status false(available)")
			return transaction_result.Error
		}

		return nil
	})
	if err != nil {
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
	var contributorIssue ContributorIssue

	manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("ASSIGN")).Str("contributor_handle", contributorHandle).Msg("Obtaining the id of the contributor from the Contributors table")
	result := manager.db.First(&contributorData, &Contributor{GithubHandle: contributorHandle})
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
	result = manager.db.Limit(1).First(&contributorIssue, &ContributorIssue{ContributorID: contributorData.ID})

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("CHECK_USER_ASSIGNED")).Str("contributor_handle", contributorHandle).Msg("Contributor is not assigned to any issue")
			return false, "", nil
		}

		manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("CHECK_USER_ASSIGNED")).Err(result.Error).Str("contributor_handle", contributorHandle).Msg("Could not obtain contributor issue assignment")
		return false, "", result.Error
	}

	// If contributor exists but issue id is 0, no issues assigned
	if contributorIssue.IssueID != 0 {
		// Querying issue the contributor is assigned to

		var issueData Issue
		issueURL := ""
		result = manager.db.Find(&issueData, Issue{ID: contributorIssue.IssueID})

		if result.Error != nil {
			manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("CHECK_USER_ASSIGNED")).Err(result.Error).Str("contributor_handle", contributorHandle).Msg("Could not find the issue the contributor is assigned to")
			issueURL = "unknown_issue"
		} else {
			manager.zeroLogger.Error().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("CHECK_USER_ASSIGNED")).Err(result.Error).Str("contributor_handle", contributorHandle).Int("contributor_issue_id", contributorIssue.IssueID).Msg("Contributor is assigned to another issue")
			issueURL = issueData.URL
		}

		return true, issueURL, nil
	} else {
		manager.zeroLogger.Info().Array("scope", zerolog.Arr().Str("DBMANAGER").Str("CHECK_USER_ASSIGNED")).Err(result.Error).Str("contributor_handle", contributorHandle).Msg("Contributor is not assigned to any issue")
		return false, "", nil
	}
}
