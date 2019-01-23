package cluster

import "fmt"

// MemberWithGivenIDAlreadyExistsError is returned when an
// attempt is made to add a member with a given ID when
// it already exists in the list of known members
type MemberWithGivenIDAlreadyExistsError struct {
	NodeID string
}

// MemberWithGivenIDDoesNotExistError is returned when an
// attempt is made to remove a member and the given ID
// is non-existent
type MemberWithGivenIDDoesNotExistError struct {
	NodeID string
}

func (e *MemberWithGivenIDAlreadyExistsError) Error() string {
	return fmt.Sprintf("Node %s already exists", e.NodeID)
}

func (e *MemberWithGivenIDDoesNotExistError) Error() string {
	return fmt.Sprintf("Node %s is not known", e.NodeID)
}
