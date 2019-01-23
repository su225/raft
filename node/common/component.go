package common

// ComponentLifecycle defines the starting and stopping
// of a component. Currently starting and destroying are
// the only two lifecycle events supported on a component
type ComponentLifecycle interface {
	// Start performs necessary initializations and
	// makes the component operational
	Start() error

	// Destroy performs necessary cleanup operations like
	// closing file handles, socket connections etc. This
	// makes the component non-operational. In other words,
	// any operation invoked on that component returns error
	Destroy() error
}
