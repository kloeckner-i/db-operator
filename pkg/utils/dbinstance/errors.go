package dbinstance

import "errors"

var (
	// ErrAlreadyExists is thrown when db instance already exists
	ErrAlreadyExists = errors.New("instance already exists")
	// ErrNotExists is thrown when db instance does not exists
	ErrNotExists = errors.New("instance does not exists")
)
