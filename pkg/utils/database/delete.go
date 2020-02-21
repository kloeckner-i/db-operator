package database

// Delete executes queries to delete database and user
func Delete(db Database, admin AdminCredentials) error {
	err := db.deleteDatabase(admin)
	if err != nil {
		return err
	}

	err = db.deleteUser(admin)
	if err != nil {
		return err
	}

	return nil
}
