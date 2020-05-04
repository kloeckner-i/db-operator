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

// New returns database interface according to engine type
func New(engine string) Database {
	switch engine {
	case "postgres":
		return &Postgres{}
	case "mysql":
		return &Mysql{}
	}

	return nil
}
