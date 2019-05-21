package protocol

import (
	"bufio"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// CommandReader ...
type CommandReader struct {
	reader *bufio.Reader
}

// NewCommandReader ...
func NewCommandReader(reader io.Reader) *CommandReader {
	return &CommandReader{
		reader: bufio.NewReader(reader),
	}
}

func (r *CommandReader) Read() (interface{}, error) {
	// Read the first part	
	commandName, err := r.reader.ReadString(' ')

	if err != nil {
		return nil, err
	}

	// Read the second part
	commandBody, err := r.reader.ReadString('\n')

	if err != nil {
		return nil, err
	}

	commandName = strings.TrimSpace(commandName)
	commandBody = strings.TrimSpace(commandBody)

	switch commandName {
	case "NEXT":
		return &ReserveCommand{}, nil

	case "DONE":
		var (
			doneRegex = regexp.MustCompile(`^(\d+)\:(\d+)$`)
			id uint64
			exitCode int
		)
		
		if match := doneRegex.FindStringSubmatch(commandBody); match != nil {
			id, err = strconv.ParseUint(match[1],10,64)

			if (err == nil) {
				exitCode, err = strconv.Atoi(match[2])
			}

			if (err == nil) {
				return &ResultCommand{ ID: id, ExitCode: exitCode, }, nil
			}
		}

		if err != nil {
			return nil, err
		}
	}

	return nil, UnknownCommand
}

// ReadAll ...
func (r *CommandReader) ReadAll() ([]interface{}, error) {
	commands := []interface{}{}

	for {
		command, err := r.Read()

		if command != nil {
			commands = append(commands, command)
		}

		if err == io.EOF {
			break
		} else if err != nil {
			return commands, err
		}
	}

	return commands, nil
}