package heartbeat

type heartbeatControllerCommand interface {
	IsHeartbeatControllerCommand() bool
}
type heartbeatControllerDestroy struct {
	heartbeatControllerCommand
	errorChannel chan error
}

type heartbeatControllerPause struct {
	heartbeatControllerCommand
	errorChannel chan error
}

type heartbeatControllerResume struct {
	heartbeatControllerCommand
	errorChannel chan error
}
