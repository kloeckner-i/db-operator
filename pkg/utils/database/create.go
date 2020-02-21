package database

// Create executes queries to create database and user
func Create(db Database, admin AdminCredentials) error {
	err := db.createDatabase(admin)
	if err != nil {
		return err
	}

	err = db.createUser(admin)
	if err != nil {
		return err
	}
	return nil
}
