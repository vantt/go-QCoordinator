package protocol

import (
	"errors"
	"strconv"
)

var (
	UnknownCommandErr = errors.New("Unknown command")
)

type CommandInterface interface {
	Send(w *CommandWriter)
}

// HelloCommand is used for sending new message from client
type HelloCommand struct {}

// ReserveCommand is used for requesting new task from the server
type ReserveCommand struct {}

type NoTaskCommand struct {	}

type UnknownCommand struct {}

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

func (c *HelloCommand) Send(w *CommandWriter) {
	w.writeString("HELLO \n")
}

func (c *NoTaskCommand) Send(w *CommandWriter) {
	w.writeString("EMPTY \n")
}

func (c *UnknownCommand) Send(w *CommandWriter) {
	w.writeString("UNKNOWN \n")
}

func (c *TaskCommand) Send(w *CommandWriter) {
	w.writeString("TASK " + c.Queue + ":" + strconv.FormatUint(c.ID, 10) + " ")
	w.writer.Write(c.Payload)
}