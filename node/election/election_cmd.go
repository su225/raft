package election

type leaderElectionManagerCommand interface {
	isLeaderElectionManagerCommand() bool
}
type leaderElectionManagerDestroy struct {
	leaderElectionManagerCommand
	errorChan chan error
}

type leaderElectionManagerPause struct {
	leaderElectionManagerCommand
	errorChan chan error
}

type leaderElectionManagerResume struct {
	leaderElectionManagerCommand
	errorChan chan error
}

type leaderElectionManagerReset struct {
	leaderElectionManagerCommand
	errorChan chan error
}
