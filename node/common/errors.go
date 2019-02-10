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
