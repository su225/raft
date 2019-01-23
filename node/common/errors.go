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

func (e *ComponentHasNotStartedError) Error() string {
	return fmt.Sprintf("Component %s has not yet started. Message=%s", e.ComponentName, e.Message)
}

func (e *ComponentIsDestroyedError) Error() string {
	return fmt.Sprintf("Component %s is destroyed. Message=%s", e.ComponentName, e.Message)
}
