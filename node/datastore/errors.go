package datastore

import "fmt"

// KeyNotFoundError is returned when the key is
// not in the datastore
type KeyNotFoundError struct {
	Key string
}

func (e *KeyNotFoundError) Error() string {
	return fmt.Sprintf("key %s not in store", e.Key)
}
