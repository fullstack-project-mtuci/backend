package repositories

import "errors"

// ErrNotFound indicates that the requested record doesn't exist.
var ErrNotFound = errors.New("record not found")
