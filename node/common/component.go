// Copyright (c) 2019 Suchith J N

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

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

// Recoverable component specifies that the state of the
// component is recoverable across crashes.
type Recoverable interface {
	// Recover recovers the state of the component
	// after a restart (happens during start)
	Recover() error
}

// Pausable component specifies that the component can temporarily
// stop accepting commands from the user or other components. If a
// component is pausable then there must be a way to continue.
type Pausable interface {
	// Pause pauses the operation of the component. If the component
	// is already paused, then this is a no-op. It doesn't affect
	// destroyed components
	Pause() error
	// Resume resumes the operation of the component unless it is
	// destroyed. If it is already running then this is a no-op
	Resume() error
}

// Freezable component specifies that the component's operation
// can be frozen temporarily. Similar to pause with one difference -
// the number of unfreeze calls must be equal to the number of freeze
// calls to make the component operational again. If there are too
// many unfreeze calls then they are ignored once the component is
// unfrozen. If there are too many freeze calls then the component
// will not be opeartional.
type Freezable interface {
	// Freeze freezes the operation of the
	// component in a reversible manner
	Freeze() error

	// Unfreeze unfreezes the component by
	// one level. If the number of unfreeze is
	// equal to the number of previous freezes
	// then component becomes operational again
	Unfreeze() error
}
