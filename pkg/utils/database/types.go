package database

// Credentials contains credentials to connect database
type Credentials struct {
	Name     string
	Username string
	Password string
}

// AdminCredentials contains admin username and password of database server
type AdminCredentials struct {
	Username string `yaml:"user"`
	Password string `yaml:"password"`
}

// Database is interface for CRUD operate of different types of databases
type Database interface {
	createDatabase(admin AdminCredentials) error
	createUser(admin AdminCredentials) error
	deleteDatabase(admin AdminCredentials) error
	deleteUser(admin AdminCredentials) error
	CheckStatus() error
	GetCredentials() Credentials
	ParseAdminCredentials(data map[string][]byte) (AdminCredentials, error)
}
