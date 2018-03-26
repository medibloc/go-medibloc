package accounts

import (
	"fmt"
)

type AuthNeededError struct {
	Needed string // Extra authentication the user needs to provide
}

// NewAuthNeededError creates a new authentication error with the extra details
// about the needed fields set.
func NewAuthNeededError(needed string) error {
	return &AuthNeededError{
		Needed: needed,
	}
}

// Error implements the standard error interface.
func (err *AuthNeededError) Error() string {
	return fmt.Sprintf("authentication needed: %s", err.Needed)
}
