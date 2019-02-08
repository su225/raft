package log

type commandType uint8

const (
	start commandType = iota
	destroy
	pause
	resume
	freeze
	unfreeze
)

type entryGCCommand struct {
	cmd     commandType
	errChan chan error
}
