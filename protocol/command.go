package protocol

import "errors"

var (
	UnknownCommand = errors.New("Unknown command")
)

// HelloCommand is used for sending new message from client
type HelloCommand struct {}

// ReserveCommand is used for requesting new task from the server
type ReserveCommand struct {}

// ResultCommand is used for notifying the server that task has been processed
type ResultCommand struct {
	ID    uint64
	ExitCode int
}

// TaskCommand is used for send the task content to client
type TaskCommand struct {
	ID      uint64
	Queue   string
	Payload []byte
}

type NoTaskCommand struct {	}