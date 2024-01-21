package models

import "errors"

var (
	ErrNoRecord      = errors.New("models: no matching record found")
	ErrInvalidCreds  = errors.New("models: invalid creds")
	ErrDuplicateMail = errors.New("models: duplicate email")
)
