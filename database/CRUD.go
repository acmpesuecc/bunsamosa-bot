package database

import (
	"errors"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"log"
)

func (manager *DBManager) Init(connection_string string) error {

	log.Println("[DBMANAGER] Initializing Database")
	// Initialize The GORM DB interface
	db, err := gorm.Open(sqlite.Open(connection_string), &gorm.Config{})
	if err != nil {
		log.Println("[ERROR][DBMANAGER] Could not initialize Database ->", err)
		return err
	} else {
		manager.db = db
		log.Println("[DBMANAGER] Successfully Initialized Database")
	}

	log.Println("[DBMANAGER] Beginning Model Automigration")

	err = manager.db.AutoMigrate(&ContributorModel{})
	if err != nil {
		log.Println("[ERROR][DBMANAGER] Could not AutoMigrate ContributorModel ->", err)
		return err
	} else {
		log.Println("[DBMANAGER] Successfully AutoMigrated ContributorModel")
	}

	err = manager.db.AutoMigrate(&ContributorRecordModel{})
	if err != nil {
		log.Println("[ERROR][DBMANAGER] Could not AutoMigrate ContributorRecordModel ->", err)
		return err
	} else {
		log.Println("[DBMANAGER] Successfully AutoMigrated ContributorRecordModel")
	}

	err = manager.db.AutoMigrate(&MaintainerModel{})
	if err != nil {
		log.Println("[ERROR][DBMANAGER] Could not AutoMigrate MaintainerModel ->", err)
		return err
	} else {
		log.Println("[DBMANAGER] Successfully AutoMigrated MaintainerModel")
	}

	err = manager.db.AutoMigrate(&Maintainer{})
	if err != nil {
		log.Println("[ERROR][DBMANAGER] Could not AutoMigrate Maintainer ->", err)
	} else {
		log.Println("[DBMANAGER] Sucessfully AutoMigrated Maintainer")
	}

	err = manager.db.AutoMigrate(&Repo{})
	if err != nil {
		log.Println("[ERROR][DBMANAGER] Could not AutoMigrate Repo ->", err)
	} else {
		log.Println("[DBMANAGER] Sucessfully AutoMigrated Repo")
	}

	err = manager.db.AutoMigrate(&MaintainerRepo{})
	if err != nil {
		log.Println("[ERROR][DBMANAGER] Could not AutoMigrate MaintainerRepo ->", err)
	} else {
		log.Println("[DBMANAGER] Sucessfully AutoMigrated MaintainerRepo")
	}

	err = manager.db.AutoMigrate(&Issue{})
	if err != nil {
		log.Println("[ERROR][DBMANAGER] Could not AutoMigrate Issue ->", err)
	} else {
		log.Println("[DBMANAGER] Sucessfully AutoMigrated Issue")
	}

	err = manager.db.AutoMigrate(&Contributor{})
	if err != nil {
		log.Println("[ERROR][DBMANAGER] Could not AutoMigrate Contributor ->", err)
	} else {
		log.Println("[DBMANAGER] Sucessfully AutoMigrated Contributor")
	}

	err = manager.db.AutoMigrate(&ContributorIssue{})
	if err != nil {
		log.Println("[ERROR][DBMANAGER] Could not AutoMigrate ContributorIssue ->", err)
	} else {
		log.Println("[DBMANAGER] Sucessfully AutoMigrated ContributorIssue")
	}

	err = manager.db.AutoMigrate(&BountyLogging{})
	if err != nil {
		log.Println("[ERROR][DBMANAGER] Could not AutoMigrate BountyLogging ->", err)
	} else {
		log.Println("[DBMANAGER] Sucessfully AutoMigrated BountyLogging")
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

	log.Println("[DBMANAGER][BOUNTY] Beginning Transaction to Assign Bounty")
	// Create the dummy record for the contributor_model
	// contributor_model := ContributorModel{name: contributor}

	// Create the time-series record of this transaction
	log.Println("[DBMANAGER][BOUNTY] Creating Contributor Record Model")

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

	log.Println("[DBMANAGER][BOUNTY] Creating Contributor Record Model -> ", crm)
	log.Println("[DBMANAGER][BOUNTY] Beginning Transaction -> ", crm)

	manager.db.Transaction(func(tx *gorm.DB) error {

		// Create the time-series record
		result := tx.Create(&crm)
		if result.Error != nil {

			// Edge Case - User record already exists in time-series data
			// In that case, update that

			log.Println("[ERROR][DBMANAGER][BOUNTY] Could Not Create ContributorRecordModel ->", result.Error)
			return result.Error
		} else {
			log.Println("[DBMANAGER][BOUNTY] Successfully Created Contributor Record Model")
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

		log.Println("[DBMANAGER][LEADERBOARD] Beginning Recompute of ContributorModel")

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
			log.Println("[ERROR][DBMANAGER][LEADERBOARD] Could Not Recompute ContributorModel ->", result.Error)
			return result.Error
		} else {
			log.Println("[DBMANAGER][LEADERBOARD] Successfully Recomputed ContributorModel")
		}
		// commit the transaction
		return nil
	})

	return nil
}

func (manager *DBManager) Get_all_records() ([]ContributorRecordModel, error) {

	// Declare the array of all records
	var records []ContributorRecordModel

	// Fetch from the database
	log.Println("[DBMANAGER|RECORDS] Fetching All Records")
	fetch_result := manager.db.Find(&records)
	if fetch_result.Error != nil {
		log.Println("[ERROR][DBMANAGER|RECORDS] Could not fetch all records ->", fetch_result.Error)
		return nil, fetch_result.Error
	} else {
		log.Println("[DBMANAGER|RECORDS] Successfully Fetched all records")
		return records, nil
	}

}

func (manager *DBManager) Get_user_records(contributor string) ([]ContributorRecordModel, error) {
	query := `select * from contributor_record_models
         where contributor_name like ?
         order by created_at desc;`

	// Declare the array of all records
	var records []ContributorRecordModel

	// Fetch from the database
	log.Println("[DBMANAGER|USER-SPECIFIC] Fetching Records for user:", contributor)

	fetch_result := manager.db.Raw(query, contributor).Scan(&records)

	if fetch_result.Error != nil {
		log.Println("[ERROR][DBMANAGER|USER-SPECIFIC] Could not fetch records for", contributor, " ->", fetch_result.Error)
		return nil, fetch_result.Error
	} else {
		log.Println("[DBMANAGER|USER-SPECIFIC] Successfully Fetched all records for user:", contributor)
		return records, nil
	}
}

func (manager *DBManager) Get_leaderboard() ([]ContributorModel, error) {

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
	log.Println("[DBMANAGER|LEADERBOARD] Fetching All Records")

	fetch_result := manager.db.Raw(leaderboard_query).Scan(&records)

	if fetch_result.Error != nil {
		log.Println("[ERROR][DBMANAGER|LEADERBOARD] Could not fetch all records ->", fetch_result.Error)
		return nil, fetch_result.Error
	} else {
		log.Println("[DBMANAGER|LEADERBOARD] Successfully Fetched all records")
		return records, nil
	}

}

func (manager *DBManager) Get_leaderboard_mat() ([]ContributorModel, error) {
	// Declare the array of all records
	var records []ContributorModel

	// Fetch from the database
	//log.Println("[DBMANAGER|MUX-LB] Fetching All Records")
	fetch_result := manager.db.Find(&records)
	if fetch_result.Error != nil {
		log.Println("[ERROR][DBMANAGER|MUX-LB] Could not fetch all records ->", fetch_result.Error)
		return nil, fetch_result.Error
	} else {
		//log.Println("[DBMANAGER|MUX-LB] Successfully Fetched all records")
		return records, nil
	}
}

func (manager *DBManager) Check_is_maintainer(user_name string) (bool, error) {
	var maintainer MaintainerModel

	log.Printf("[DBMANAGER|CHECK] Checking if %s is a maintainer\n", user_name)
	result := manager.db.Limit(1).First(&maintainer, "username like ?", user_name)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			log.Printf("[DBMANAGER|CHECK] %s IS NOT a maintainer\n", user_name)
			return false, nil
		}
		log.Println("[ERROR][DBMANAGER|CHECK] Could not check maintainer ->", result.Error)
		return false, result.Error
	}

	log.Printf("[DBMANAGER|CHECK] %s IS a maintainer\n", user_name)
	return true, nil
}

// TODO: Add status and repo id while creating record
// Check status and closure status before assigning issue
func (manager *DBManager) AssignIssue(issueURL string, contributorHandle string) (bool, error) {
	// Get issue_id from Issues table, create the issue record if it does not exist
	// Get the contributor_id from the Contributors table, create the contributor
	// record if it does not exist Check if the contributor has not been assigned
	// another issue Check if the issue has not been assigned to another
	// contributor If both the checks yield true, update the record in the
	// ContributorIssues table
	var issueData Issue
	var contributorData Contributor

	log.Printf("[DBMANAGER|ASSIGN] Obtaining the id of issue %q from the Issues table\n", issueURL)
	// Fetch the record with matching conditions or create a new record
	result := manager.db.FirstOrCreate(&issueData, &Issue{URL: issueURL})
	if result.Error != nil {
		log.Printf("[ERROR][DBMANAGER|ASSIGN] Could not obtain issue %q from the Issues table", issueURL)
		return false, result.Error
	}

	log.Printf("[DBMANAGER|ASSIGN] Obtaining the id of contributor %q from the Contributors table\n", contributorHandle)
	result = manager.db.FirstOrCreate(&contributorData, &Contributor{GithubHandle: contributorHandle})
	if result.Error != nil {
		log.Printf("[ERROR][DBMANAGER|ASSIGN] Could not obtain contributor %q from the Contributors table", contributorHandle)
		return false, result.Error
	}

	log.Printf("[DBMANAGER|ASSIGN] Checking if contributor %q has already been assigned an issue\n", contributorHandle)
	var contributorIssue ContributorIssue
	result = manager.db.Find(&contributorIssue, "contributor_id = ?", contributorData.ID)

	if result.Error != nil {
		// If the error is a missing record, continue to assign the issue and add a record to the table
		// Else, return the error
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			log.Printf("[ERROR][DBMANAGER|ASSIGN] Could not query contributor %q with contributor id %d from the ContributorIssues table\n", contributorHandle, contributorData.ID)
			return false, result.Error
		}
	} else {
		// If contributor is assigned another issue (IssueID != 0), return false without an error
		if contributorIssue.IssueID != 0 {
			log.Printf("[DBMANAGER|ASSIGN] Contributor %q with ContributorId %d has already been assigned an issue with IssueId %d\n", contributorHandle, contributorData.ID, issueData.IssueID)
			return false, nil
		}
	}

	log.Printf("[DBMANAGER|ASSIGN] Storing assignment of issue with IssueId %d to contributor %q with ContributorID %d\n", issueData.IssueID, contributorHandle, contributorData.ID)
	result = manager.db.FirstOrCreate(&contributorIssue, ContributorIssue{IssueID: issueData.IssueID})
	if result.Error != nil {
		log.Printf("[ERROR][DBMANAGER|ASSIGN] Could not obtain issue with IssueId %d from the ContributorIssues table", issueData.IssueID)
		return false, result.Error
	}

	contributorIssue.ContributorID = contributorData.ID
	result = manager.db.Save(&contributorIssue)
	if result.Error != nil {
		log.Printf("[ERROR][DBMANAGER|ASSIGN] Could not store assignment of issue with IssueId %d to contributor %q with ContributorID %d\n", issueData.IssueID, contributorHandle, contributorData.ID)
		return false, result.Error
	}

	return true, nil
}

func (manager *DBManager) DeassignIssue(issueURL string) (bool, error) {
	var issueData Issue
	var contributorIssue ContributorIssue

	log.Printf("[DBMANAGER|DEASSIGN] Obtaining the id of issue %q from the Issues table\n", issueURL)
	// Fetch the issue record from the Issues table
	result := manager.db.First(&issueData, "url LIKE ?", issueURL)
	if result.Error != nil {
		// If the issue is not found, log and return false without an error, or modify if u want error
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			log.Printf("[DBMANAGER|DEASSIGN] Issue %q not found in the Issues table\n", issueURL)
			return false, nil
		}
		log.Printf("[ERROR][DBMANAGER|DEASSIGN] Could not obtain issue %q from the Issues table", issueURL)
		return false, result.Error
	}

	log.Printf("[DBMANAGER|DEASSIGN] Checking if the issue with IssueId %d has been assigned to any contributor\n", issueData.IssueID)
	// Fetch the contributor issue record from the ContributorIssues table
	result = manager.db.Find(&contributorIssue, "issue_id = ?", issueData.IssueID)
	if result.Error != nil {
		// If no contributor is assigned to the issue, log and return false without an error
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			log.Printf("[DBMANAGER|DEASSIGN] No contributor assigned to issue %q\n", issueURL)
			return false, nil
		}
		log.Printf("[ERROR][DBMANAGER|DEASSIGN] Could not query issue %q with IssueId %d from the ContributorIssues table\n", issueURL, issueData.IssueID)
		return false, result.Error
	}

	log.Printf("[DBMANAGER|DEASSIGN] Removing assignment of issue with IssueId %d from contributor with ContributorID %d\n", issueData.IssueID, contributorIssue.ContributorID)
	// Start a new transaction
	err := manager.db.Transaction(func(tx *gorm.DB) error {
		// Set the contributor ID to 0 in the ContributorIssues table
		contributorIssue.ContributorID = 0
		result = tx.Save(&contributorIssue)
		if result.Error != nil {
			log.Printf("[ERROR][DBMANAGER|DEASSIGN] Could not remove assignment of issue with IssueId %d from contributor with ContributorID %d\n", issueData.IssueID, contributorIssue.ContributorID)
			return result.Error
		}

		// Update the status of the issue, change to bool later
		issueData.Status = "unassigned"
		result = tx.Save(&issueData)
		if result.Error != nil {
			log.Printf("[ERROR][DBMANAGER|DEASSIGN] Could not update status of issue with IssueId %d\n", issueData.IssueID)
			return result.Error
		}

		return nil
	})

	if err != nil {
		return false, err
	}

	return true, nil
}
func (manager *DBManager) WithdrawIssue(issueURL string) {}
func (manager *DBManager) ExtendIssue(issueURL string)   {}
