package dbinstance

// AdminCredentials is superuser and password of database server
type AdminCredentials struct {
	Username string `yaml:"user"`
	Password string `yaml:"password"`
}

// DbInstance interface to operate database server
type DbInstance interface {
	exist() error
	create() error
	update() error
	getInfoMap() (map[string]string, error)
}
