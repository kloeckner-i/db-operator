package dbinstance

// DbInstance interface to operate database server
type DbInstance interface {
	exist() error
	create() error
	update() error
	getInfoMap() (map[string]string, error)
}
