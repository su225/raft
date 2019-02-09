package common

import "fmt"

// ComponentHasNotStartedError occurs when some operation (which is
// usually not associated with component's lifecycle) is invoked on
// a component when it has not yet started
type ComponentHasNotStartedError struct {
	ComponentName string
	Message       string
}

// ComponentIsDestroyedError occurs when some operation (which is
// usually not associated with component's lifecycle) is invoked on
// a component that is already destroyed
type ComponentIsDestroyedError struct {
	ComponentName string
	Message       string
}

// ComponentIsPausedError is invoked when an operation is invoked
// and it requires the component to be functional and running
type ComponentIsPausedError struct {
	ComponentName string
	Message       string
}

// ComponentIsFrozenError occurs when an attempt is made to perform
// some operation on a frozen component (other than freezing/unfreezing)
// In frozen state, read-only operations are permitted though.
type ComponentIsFrozenError struct {
	ComponentName string
	Message       string
}

func (e *ComponentHasNotStartedError) Error() string {
	return fmt.Sprintf("Component %s has not yet started. Message=%s", e.ComponentName, e.Message)
}

func (e *ComponentIsDestroyedError) Error() string {
	return fmt.Sprintf("Component %s is destroyed. Message=%s", e.ComponentName, e.Message)
}

func (e *ComponentIsPausedError) Error() string {
	return fmt.Sprintf("Component %s is paused. Message=%s", e.ComponentName, e.Message)
}

func (e *ComponentIsFrozenError) Error() string {
	return fmt.Sprintf("Component %s is frozen. Message=%s", e.ComponentName, e.Message)
}
