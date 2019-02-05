package snapshot

// SnapshotMetadata consists of epoch and
// the index upto which snapshot is taken
type SnapshotMetadata struct {
	SnapshotIndex uint64
	Epoch         uint64
}

// SnapshotHandler is responsible for taking
// snapshot of data store. It supports operations like
// adding/removing key or iterating through entries to
// create add/remove operations by itself
type SnapshotHandler struct {
}
