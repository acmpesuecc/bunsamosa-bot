package database

import (
	"errors"

	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func (manager *DBManager) Init(connection_string string, sugaredLogger *zap.SugaredLogger) error {
	manager.sugaredLogger = sugaredLogger

	manager.sugaredLogger.Infof("Initializing Database",
		zap.Strings("scope", []string{"DBMANAGER"}),
	)

	// Initialize The GORM DB interface
	db, err := gorm.Open(sqlite.Open(connection_string), &gorm.Config{})
	if err != nil {

		manager.sugaredLogger.Panicw("Could not initialize Database ->", err,
			zap.Strings("scope", []string{"DBMANAGER"}),
		)

		return err
	} else {
		manager.db = db
		manager.sugaredLogger.Infof("Successfully Initialized Database",
			zap.Strings("scope", []string{"DBMANAGER"}),
		)
	}

	manager.sugaredLogger.Infof("Beginning Model Automigration",
		zap.Strings("scope", []string{"DBMANAGER"}),
	)

	err = manager.db.AutoMigrate(&ContributorModel{})
	if err != nil {
		manager.sugaredLogger.Errorf("Could not AutoMigrate ContributorModel ->", err,
			zap.Strings("scope", []string{"DBMANAGER"}),
		)
		return err
	} else {
		manager.sugaredLogger.Infof("Successfully AutoMigrated ContributorModel",
			zap.Strings("scope", []string{"DBMANAGER"}),
		)
	}

	err = manager.db.AutoMigrate(&ContributorRecordModel{})
	if err != nil {
		manager.sugaredLogger.Errorf("Could not AutoMigrate ContributorRecordModel ->", err,
			zap.Strings("scope", []string{"DBMANAGER"}),
		)
		return err
	} else {
		manager.sugaredLogger.Infof("Successfully AutoMigrated ContributorRecordModel",
			zap.Strings("scope", []string{"DBMANAGER"}),
		)
	}

	err = manager.db.AutoMigrate(&MaintainerModel{})
	if err != nil {
		manager.sugaredLogger.Errorf("Could not AutoMigrate MaintainerModel ->", err,
			zap.Strings("scope", []string{"DBMANAGER"}),
		)
		return err
	} else {
		manager.sugaredLogger.Infof("Successfully AutoMigrated MaintainerModel",
			zap.Strings("scope", []string{"DBMANAGER"}),
		)
	}

	err = manager.db.AutoMigrate(&Maintainer{})
	if err != nil {
		manager.sugaredLogger.Errorf("Could not AutoMigrate Maintainer ->", err,
			zap.String("scope", "DBMANAGEER"),
		)
	} else {
		manager.sugaredLogger.Infof("Sucessfully AutoMigrated Maintainer",
			zap.Strings("scope", []string{"DBMANAGER"}),
		)
	}

	err = manager.db.AutoMigrate(&Repo{})
	if err != nil {
		manager.sugaredLogger.Errorf("Could not AutoMigrate Repo ->", err,
			zap.Strings("scope", []string{"DBMANAGER"}),
		)
	} else {
		manager.sugaredLogger.Infof("Sucessfully AutoMigrated Repo",
			zap.Strings("scope", []string{"DBMANAGER"}),
		)
	}

	err = manager.db.AutoMigrate(&MaintainerRepo{})
	if err != nil {
		manager.sugaredLogger.Errorf("Could not AutoMigrate MaintainerRepo ->", err,
			zap.Strings("scope", []string{"DBMANAGER"}),
		)
	} else {
		manager.sugaredLogger.Infof("Sucessfully AutoMigrated MaintainerRepo",
			zap.Strings("scope", []string{"DBMANAGER"}),
		)
	}

	err = manager.db.AutoMigrate(&Issue{})
	if err != nil {
		manager.sugaredLogger.Errorf("Could not AutoMigrate Issue ->", err,
			zap.Strings("scope", []string{"DBMANAGER"}),
		)
	} else {
		manager.sugaredLogger.Infof("Sucessfully AutoMigrated Issue",
			zap.Strings("scope", []string{"DBMANAGER"}),
		)
	}

	err = manager.db.AutoMigrate(&Contributor{})
	if err != nil {
		manager.sugaredLogger.Errorf("Could not AutoMigrate Contributor ->", err,
			zap.Strings("scope", []string{"DBMANAGER"}),
		)
	} else {
		manager.sugaredLogger.Infof("Sucessfully AutoMigrated Contributor",
			zap.Strings("scope", []string{"DBMANAGER"}),
		)
	}

	err = manager.db.AutoMigrate(&ContributorIssue{})
	if err != nil {
		manager.sugaredLogger.Errorf("Could not AutoMigrate ContributorIssue ->", err,
			zap.Strings("scope", []string{"DBMANAGER"}),
		)
	} else {
		manager.sugaredLogger.Infof("Sucessfully AutoMigrated ContributorIssue",
			zap.Strings("scope", []string{"DBMANAGER"}),
		)
	}

	err = manager.db.AutoMigrate(&BountyLogging{})
	if err != nil {
		manager.sugaredLogger.Errorf("Could not AutoMigrate BountyLogging ->", err,
			zap.Strings("scope", []string{"DBMANAGER"}),
		)
	} else {
		manager.sugaredLogger.Infof("Sucessfully AutoMigrated BountyLogging",
			zap.Strings("scope", []string{"DBMANAGER"}),
		)
	}

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

	manager.sugaredLogger.Infof("Beginning Transaction to Assign Bounty",
		zap.Strings("scope", []string{"DBMANAGER", "BOUNTY"}),
	)
	// Create the dummy record for the contributor_model
	// contributor_model := ContributorModel{name: contributor}

	// Create the time-series record of this transaction
	manager.sugaredLogger.Infof("Creating Contributor Record Model",
		zap.Strings("scope", []string{"DBMANAGER", "BOUNTY"}),
	)

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

	manager.sugaredLogger.Infof("Creating Contributor Record Model -> ", crm,
		zap.Strings("scope", []string{"DBMANAGER", "BOUNTY"}),
	)
	manager.sugaredLogger.Infof("Beginning Transaction -> ", crm,
		zap.Strings("scope", []string{"DBMANAGER", "BOUNTY"}),
	)

	manager.db.Transaction(func(tx *gorm.DB) error {

		// Create the time-series record
		result := tx.Create(&crm)
		if result.Error != nil {

			// Edge Case - User record already exists in time-series data
			// In that case, update that

			manager.sugaredLogger.Errorf("Could Not Create ContributorRecordModel ->", result.Error,
				zap.Strings("scope", []string{"DBMANAGER", "BOUNTY"}),
			)
			return result.Error
		} else {
			manager.sugaredLogger.Infof("Successfully Created Contributor Record Model",
				zap.Strings("scope", []string{"DBMANAGER", "BOUNTY"}),
			)
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

		manager.sugaredLogger.Infof("Beginning Recompute of ContributorModel",
			zap.Strings("scope", []string{"DBMANAGER", "LEADERBOARD"}),
		)

		//Recompute ContributorModel Table
		lb_query := `DELETE FROM contributor_models;INSERT INTO contributor_models (Name, Current_bounty)
SELECT contributor_name AS Name, sum(latest_points) AS Current_bounty from (
   select
       contributor_name, (SELECT points_allotted FROM contributor_record_models where t1.pullreq_url = pullreq_url order by created_at desc limit 1) as latest_points
   from contributor_record_models as t1
   GROUP by pullreq_url, contributor_name
) GROUP BY contributor_name;`

		result = tx.Exec(lb_query)
		if result.Error != nil {
			manager.sugaredLogger.Errorf("Could Not Recompute ContributorModel ->", result.Error,
				zap.Strings("scope", []string{"DBMANAGER", "LEADERBOARD"}),
			)
			return result.Error
		} else {
			manager.sugaredLogger.Infof("Successfully Recomputed ContributorModel",
				zap.Strings("scope", []string{"DBMANAGER", "LEADERBOARD"}),
			)
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
	manager.sugaredLogger.Infof("Fetching All Records",
		zap.Strings("scope", []string{"DBMANAGER", "RECORDS"}),
	)
	fetch_result := manager.db.Find(&records)
	if fetch_result.Error != nil {
		manager.sugaredLogger.Errorf("Could not fetch all records ->", fetch_result.Error,
			zap.Strings("scope", []string{"DBMANAGER", "RECORDS"}),
		)
		return nil, fetch_result.Error
	} else {
		manager.sugaredLogger.Infof("Successfully Fetched all records",
			zap.Strings("scope", []string{"DBMANAGER", "RECORDS"}),
		)

		return records, nil
	}

}

func (manager *DBManager) GetUserRecords(contributor string) ([]ContributorRecordModel, error) {
	query := `select * from contributor_record_models
         where contributor_name like ?
         order by created_at desc;`

	// Declare the array of all records
	var records []ContributorRecordModel

	// Fetch from the database
	manager.sugaredLogger.Infof("[DBMANAGER|USER-SPECIFIC] Fetching Records for user:", contributor,
		zap.Strings("scope", []string{"DBMANAGER", "USER-SPECIFIC"}),
	)

	fetch_result := manager.db.Raw(query, contributor).Scan(&records)

	if fetch_result.Error != nil {
		manager.sugaredLogger.Errorf("Could not fetch records for", contributor, " ->", fetch_result.Error,
			zap.Strings("scope", []string{"DBMANAGER", "USER-SPECIFIC"}),
		)

		return nil, fetch_result.Error
	} else {
		manager.sugaredLogger.Infof("[DBMANAGER|USER-SPECIFIC] Successfully Fetched all records for user:", contributor,
			zap.Strings("scope", []string{"DBMANAGER", "USER-SPECIFIC"}),
		)
		return records, nil
	}
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
	manager.sugaredLogger.Infof("Fetching All Records",
		zap.Strings("scope", []string{"DBMANAGER", "LEADERBOARD"}),
	)

	fetch_result := manager.db.Raw(leaderboard_query).Scan(&records)

	if fetch_result.Error != nil {
		manager.sugaredLogger.Errorf("Could not fetch all records ->", fetch_result.Error,
			zap.Strings("scope", []string{"DBMANAGER", "LEADERBOARD"}),
		)
		return nil, fetch_result.Error
	} else {
		manager.sugaredLogger.Infof("[DBMANAGER|LEADERBOARD] Successfully Fetched all records",
			zap.Strings("scope", []string{"DBMANAGER", "LEADERBOARD"}),
		)

		return records, nil
	}

}

func (manager *DBManager) GetLeaderboardMat() ([]ContributorModel, error) {
	// Declare the array of all records
	var records []ContributorModel

	// Fetch from the database
	//manager.sugaredLogger.Infof("[DBMANAGER|MUX-LB] Fetching All Records")
	fetch_result := manager.db.Find(&records)
	if fetch_result.Error != nil {
		manager.sugaredLogger.Errorf("Could not fetch all records ->", fetch_result.Error,
			zap.Strings("scope", []string{"DBMANAGER", "MUX-LB"}),
		)
		return nil, fetch_result.Error
	} else {
		manager.sugaredLogger.Infof("Successfully Fetched all records",
			zap.Strings("scope", []string{"DBMANAGER", "MUX-LB"}),
		)
		return records, nil
	}
}

func (manager *DBManager) CheckIsMaintainer(user_name string) (bool, error) {
	var maintainer MaintainerModel

	manager.sugaredLogger.Infof("Checking if %s is a maintainer\n", user_name,
		zap.Strings("scope", []string{"DBMANAGER", "CHECK_MAINTAINER"}),
	)

	result := manager.db.Limit(1).First(&maintainer, "username like ?", user_name)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			manager.sugaredLogger.Infof("%s IS NOT a maintainer\n", user_name,
				zap.Strings("scope", []string{"DBMANAGER", "CHECK_MAINTAINER"}),
			)
			return false, nil
		}
		manager.sugaredLogger.Errorf("Could not check maintainer ->", result.Error,
			zap.Strings("scope", []string{"DBMANAGER", "CHECK_MAINTAINER"}),
		)
		return false, result.Error
	}

	manager.sugaredLogger.Infof("%s IS a maintainer\n", user_name,
		zap.Strings("scope", []string{"DBMANAGER", "CHECK_MAINTAINER"}),
	)
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

	manager.sugaredLogger.Infof("Obtaining the id of repo %q from the Repos table\n", repoURL,
		zap.Strings("scope", []string{"DBMANAGER", "ASSIGN"}),
	)
	// Fetch the record with matching conditions or create a new record
	result := manager.db.FirstOrCreate(&repoData, &Repo{URL: repoURL})
	if result.Error != nil {
		manager.sugaredLogger.Errorf("Could not obtain repo %q from the Repos table", repoURL,
			zap.Strings("scope", []string{"DBMANAGER", "ASSIGN"}),
		)
		return false, result.Error
	}

	manager.sugaredLogger.Infof("Obtaining the id of issue %q from the Issues table\n", issueURL,
		zap.Strings("scope", []string{"DBMANAGER", "ASSIGN"}),
	)
	// Fetch the record with matching conditions or create a new record
	result = manager.db.FirstOrCreate(&issueData, &Issue{URL: issueURL, RepoID: repoData.ID})
	if result.Error != nil {
		manager.sugaredLogger.Errorf("Could not obtain issue %q from the Issues table", issueURL,
			zap.Strings("scope", []string{"DBMANAGER", "ASSIGN"}),
		)
		return false, result.Error
	}

	manager.sugaredLogger.Infof("Obtaining the id of contributor %q from the Contributors table\n", contributorHandle,
		zap.Strings("scope", []string{"DBMANAGER", "ASSIGN"}),
	)
	result = manager.db.FirstOrCreate(&contributorData, &Contributor{GithubHandle: contributorHandle})
	if result.Error != nil {
		manager.sugaredLogger.Errorf("Could not obtain contributor %q from the Contributors table", contributorHandle,
			zap.Strings("scope", []string{"DBMANAGER", "ASSIGN"}),
		)

		return false, result.Error
	}

	manager.sugaredLogger.Infof("Checking if contributor %q has already been assigned an issue\n", contributorHandle,
		zap.Strings("scope", []string{"DBMANAGER", "ASSIGN"}),
	)
	var contributorIssue ContributorIssue
	result = manager.db.Find(&contributorIssue, "contributor_id = ?", contributorData.ID)

	if result.Error != nil {
		// If the error is a missing record, continue to assign the issue and add a record to the table
		// Else, return the error
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			manager.sugaredLogger.Errorf("Could not query contributor %q with contributor id %d from the ContributorIssues table\n", contributorHandle, contributorData.ID,
				zap.Strings("scope", []string{"DBMANAGER", "ASSIGN"}),
			)
			return false, result.Error
		}
	} else {
		// If contributor is assigned another issue (IssueID != 0), return false without an error
		if contributorIssue.IssueID != 0 {
			manager.sugaredLogger.Infof("Contributor %q with ContributorId %d has already been assigned an issue with IssueId %d\n", contributorHandle, contributorData.ID, issueData.ID,
				zap.Strings("scope", []string{"DBMANAGER", "ASSIGN"}),
			)
			return false, nil
		}
	}

	manager.sugaredLogger.Infof("Storing assignment of issue with IssueId %d to contributor %q with ContributorID %d\n", issueData.ID, contributorHandle, contributorData.ID,
		zap.Strings("scope", []string{"DBMANAGER", "ASSIGN"}),
	)

	result = manager.db.FirstOrCreate(&contributorIssue, ContributorIssue{IssueID: issueData.ID})
	if result.Error != nil {
		manager.sugaredLogger.Errorf("Could not obtain issue with IssueId %d from the ContributorIssues table", issueData.ID,
			zap.Strings("scope", []string{"DBMANAGER", "ASSIGN"}),
		)
		return false, result.Error
	}

	err := manager.db.Transaction(func(tx *gorm.DB) error {
		contributorIssue.ContributorID = contributorData.ID
		result = tx.Save(&contributorIssue)
		if result.Error != nil {
			manager.sugaredLogger.Errorf("Could not store assignment of issue with IssueId %d to contributor %q with ContributorID %d\n", issueData.ID, contributorHandle, contributorData.ID,
				zap.Strings("scope", []string{"DBMANAGER", "ASSIGN", "TRANSACTION"}),
			)
			return result.Error
		}

		issueData.Status = true
		result = tx.Save(&issueData)
		if result.Error != nil {
			manager.sugaredLogger.Errorf("[ERROR][DBMANAGER|ASSIGN] Could not store assignment of issue with IssueId %d to contributor %q with ContributorID %d\n", issueData.ID, contributorHandle, contributorData.ID,
				zap.Strings("scope", []string{"DBMANAGER", "ASSIGN", "TRANSACTION"}),
			)
			return result.Error
		}

		return nil
	})

	if err != nil {
		manager.sugaredLogger.Errorf("Could not store assignment of issue with IssueId %d to contributor %q with ContributorID %d\n", issueData.ID, contributorHandle, contributorData.ID,
			zap.Strings("scope", []string{"DBMANAGER", "ASSIGN", "TRANSACTION"}),
		)
		return false, err
	}

	return true, nil
}

func (manager *DBManager) DeassignIssue(issueURL string) (bool, error) {
	var issueData Issue
	var contributorIssue ContributorIssue

	manager.sugaredLogger.Infof("Obtaining the id of issue %q from the Issues table\n", issueURL,
		zap.Strings("scope", []string{"DBMANAGER", "DEASSIGN"}),
	)

	// Fetch the issue record from the Issues table
	result := manager.db.First(&issueData, "url LIKE ?", issueURL)
	if result.Error != nil {
		// If the issue is not found, log and return false without an error, or modify if u want error
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			manager.sugaredLogger.Infof("Issue %q not found in the Issues table\n", issueURL,
				zap.Strings("scope", []string{"DBMANAGER", "DEASSIGN"}),
			)
			return false, nil
		}
		manager.sugaredLogger.Errorf("Could not obtain issue %q from the Issues table", issueURL,
			zap.Strings("scope", []string{"DBMANAGER", "DEASSIGN"}),
		)
		return false, result.Error
	}

	manager.sugaredLogger.Infof("Checking if the issue with IssueId %d has been assigned to any contributor\n", issueData.ID,
		zap.Strings("scope", []string{"DBMANAGER", "DEASSIGN"}),
	)

	// Fetch the contributor issue record from the ContributorIssues table
	result = manager.db.Find(&contributorIssue, "issue_id = ?", issueData.ID)
	if result.Error != nil {
		// If no contributor is assigned to the issue, log and return false without an error
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			manager.sugaredLogger.Infof("No contributor assigned to issue %q\n",
				issueURL,
				zap.Strings("scope", []string{"DBMANAGER", "DEASSIGN"}),
			)
			return false, nil
		}
		manager.sugaredLogger.Errorf("Could not query issue %q with IssueId %d from the ContributorIssues table\n", issueURL, issueData.ID,
			zap.Strings("scope", []string{"DBMANAGER", "DEASSIGN"}),
		)
		return false, result.Error
	}

	manager.sugaredLogger.Infof("Removing assignment of issue with IssueId %d from contributor with ContributorID %d\n",
		issueData.ID, contributorIssue.ContributorID,
		zap.Strings("scope", []string{"DBMANAGER", "DEASSIGN"}),
	)

	// Start a new transaction
	err := manager.db.Transaction(func(tx *gorm.DB) error {
		// Set the contributor ID to 0 in the ContributorIssues table
		contributorIssue.ContributorID = 0
		result = tx.Save(&contributorIssue)
		if result.Error != nil {
			manager.sugaredLogger.Errorf("Could not remove assignment of issue with IssueId %d from contributor with ContributorID %d\n", issueData.ID, contributorIssue.ContributorID,
				zap.Strings("scope", []string{"DBMANAGER", "DEASSIGN", "TRANSACTION"}),
			)
			return result.Error
		}

		// Update the status of the issue, change to bool later
		issueData.Status = false
		result = tx.Save(&issueData)
		if result.Error != nil {
			manager.sugaredLogger.Errorf("Could not update status of issue with IssueId %d\n", issueData.ID,
				zap.Strings("scope", []string{"DBMANAGER", "DEASSIGN", "TRANSACTION"}),
			)
			return result.Error
		}

		return nil
	})

	if err != nil {
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

	manager.sugaredLogger.Infof("Obtaining the id of the issue from Issues table", zap.Strings("scope", []string{"DBMANAGER", "WITHDRAW"}))
	result := manager.db.First(&issueData, &Issue{URL: issueURL})
	if result.Error != nil {
		manager.sugaredLogger.Errorf("Could not obtain the id of issue %q from Issues table", issueURL,
			zap.Strings("scope", []string{"DBMANAGER", "WITHDRAW"}),
		)
		return false, result.Error
	}

	manager.sugaredLogger.Infof("Obtaining the id of the contributor for the contributor handle",
		zap.Strings("scope", []string{"DBMANAGER", "WITHDRAW"}),
	)
	result = manager.db.First(&contributorData, &Contributor{GithubHandle: contributorHandle})
	if result.Error != nil {
		manager.sugaredLogger.Errorf("Couldn't obtain the id of contributor %q from Contributor table", contributorHandle,
			zap.Strings("scope", []string{"DBMANAGER", "WITHDRAW"}),
		)
		return false, result.Error
	}

	manager.sugaredLogger.Infof("Checking if the contributor is assigned to the request issue",
		zap.Strings("scope", []string{"DBMANAGER", "WITHDRAW"}),
	)
	var contributorIssueData ContributorIssue
	result = manager.db.First(&contributorIssueData, "contributor_id = ?", contributorData.ID)
	if result.Error != nil {
		manager.sugaredLogger.Errorf("Could not query contributor %q with contributor id %d from the ContributorIssues table\n", contributorHandle, contributorData.ID,
			zap.Strings("scope", []string{"DBMANAGER", "WITHDRAW"}),
		)
		return false, result.Error
	} else {
		// If contributor is assigned another issue, and there is a mismatch between
		// the issue withdraw has been requested from and what they have been assigned to
		if contributorIssueData.IssueID != issueData.ID {
			manager.sugaredLogger.Infof("Contributor %q with ContributorId %d requested for a deassign on a different Issue with IssueId %d \n", contributorHandle, contributorData.ID, issueData.ID,
				zap.Strings("scope", []string{"DBMANAGER", "WITHDRAW"}),
			)
			return false, nil
		}
	}

	err := manager.db.Transaction(func(tx *gorm.DB) error {
		contributorIssueData.IssueID = 0
		transaction_result := tx.Save(&contributorIssueData)
		if transaction_result.Error != nil {
			manager.sugaredLogger.Errorf("Failed to update contributorIssueData.IssueID for contributor %q for issueID %q\n", contributorHandle, issueData.ID,
				zap.Strings("scope", []string{"DBMANAGER", "WITHDRAW", "TRANSACTION"}),
			)
			return transaction_result.Error
		}

		issueData.Status = false
		transaction_result = tx.Save(&issueData)
		if transaction_result.Error != nil {
			manager.sugaredLogger.Errorf("Failed to update IssueID %q(%q) status to false(available)\n", issueData.ID, issueURL,
				zap.Strings("scope", []string{"DBMANAGER", "WITHDRAW", "TRANSACTION"}),
			)
			return transaction_result.Error
		}

		return nil
	})

	if err != nil {
		manager.sugaredLogger.Errorf("Couldn't perform transaction to withdraw contributor %q from issue %q\n", contributorHandle, issueURL,
			zap.Strings("scope", []string{"DBMANAGER", "WITHDRAW", "TRANSACTION"}),
		)
		return false, err
	}

	return true, nil
}

func (manager *DBManager) ExtendIssue(issueURL string) {}
