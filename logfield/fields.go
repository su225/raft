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

package logfield

const (
	// Component is the component field in the logger like
	// Heartbeat, ElectionMgr and so on. The value of the
	// field must be in CAPS as a convention
	Component = "component"

	// Event is the event during which the log entry is written
	// The value of the field must be in CAPS as a convention
	Event = "event"

	// ErrorReason is used with error logger to denote the reason for error
	ErrorReason = "reason"

	// RequesterIPAddress is used in API server to log the address
	// of the origin of the request.
	RequesterIPAddress = "ip-addr"

	// RESTMethod is used in API server to log the HTTP method of the request
	// (like GET, PUT, POST, DELETE, PATCH etc)
	RESTMethod = "method"

	// RequestURI is used in API server to log the request URI
	RequestURI = "uri"
)
