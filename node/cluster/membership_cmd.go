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

package cluster

// membershipCommand represents the command to
// membership manager
type membershipCommand interface {
	isMembershipCommand() bool
}

type startMembershipManager struct {
	membershipCommand
	currentNodeInfo NodeInfo
	errChan         chan error
}

type destroyMembershipManager struct {
	membershipCommand
	errChan chan error
}

type addNode struct {
	membershipCommand
	nodeInfo NodeInfo
	errChan  chan error
}

type removeNode struct {
	membershipCommand
	nodeID  string
	errChan chan error
}

type getNode struct {
	membershipCommand
	nodeID    string
	replyChan chan *getNodeReply
}

type getNodeReply struct {
	nodeInfo NodeInfo
	err      error
}

type getAllNodes struct {
	membershipCommand
	replyChan chan []NodeInfo
}
