package protocol

import (	
	"io"	
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

func (w *CommandWriter) WriteCommand(cmd CommandInterface) {
	cmd.Send(w)
}