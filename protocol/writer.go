package protocol

import (	
	"io"
	"strconv"
)


// CommandWriter ...
type CommandWriter struct {
	writer io.Writer
}

// NewCommandWriter ... 
func NewCommandWriter(writer io.Writer) *CommandWriter {
	return &CommandWriter{
		writer: writer,
	}
}

func (w *CommandWriter) writeString(msg string) error {
	_, err := w.writer.Write([]byte(msg))

	return err
}

func (w *CommandWriter) WriteCommand(command interface{}) error {
	// naive implementation ...
	var err error

	switch v := command.(type) {
	case *HelloCommand:		
		w.writeString("HELLO \n")
		
	case *TaskCommand:		
		w.writeString("TASK " + v.Queue + ":" + strconv.FormatUint(v.ID, 10) + " ")		
		w.writer.Write(v.Payload)

	case *NoTaskCommand:		
		w.writeString("EMPTY \n")

	default:		
		err = UnknownCommand
	}

	return err
}