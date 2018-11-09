package cache

import "fmt"

// NoSuchSourceFileError is returned when attempting to access a source file
// that doesn't exist
type NoSuchSourceFileError string

func (err NoSuchSourceFileError) Error() string {
	return fmt.Sprintf("file %q does not exist", string(err))
}
