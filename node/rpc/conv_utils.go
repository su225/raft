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
