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

package rpc

import (
	"github.com/su225/raft/node/log"
	"github.com/su225/raft/pb"
)

// ConvertEntryToProtobuf converts the entry from in-memory format
// to the format as defined in protocol-buffer
func ConvertEntryToProtobuf(entry log.Entry) *raftpb.OpEntry {
	switch e := entry.(type) {
	case *log.UpsertEntry:
		return &raftpb.OpEntry{
			TermId: e.GetTermID(),
			Entry: &raftpb.OpEntry_Upsert{
				Upsert: &raftpb.UpsertOp{
					Data: &raftpb.Data{
						Key:   e.Key,
						Value: e.Value,
					},
				},
			},
		}
	case *log.DeleteEntry:
		return &raftpb.OpEntry{
			TermId: entry.GetTermID(),
			Entry: &raftpb.OpEntry_Delete{
				Delete: &raftpb.DeleteOp{
					Key: e.Key,
				},
			},
		}
	}
	return &raftpb.OpEntry{Entry: nil}
}

// ConvertProtobufToEntry converts from protobuf wire format
// to the in-memory format.
func ConvertProtobufToEntry(entry *raftpb.OpEntry) log.Entry {
	termID := entry.GetTermId()
	switch e := entry.Entry.(type) {
	case *raftpb.OpEntry_Upsert:
		return &log.UpsertEntry{
			TermID: termID,
			Key:    e.Upsert.Data.Key,
			Value:  e.Upsert.Data.Value,
		}
	case *raftpb.OpEntry_Delete:
		return &log.DeleteEntry{
			TermID: termID,
			Key:    e.Delete.Key,
		}
	}
	return &log.SentinelEntry{}
}
