package tcpserver

import (
	"net"	
	"time"	
	"github.com/vantt/go-QCoordinator/protocol"
	
)

type client struct {
	index  int
	conn   net.Conn	
	writer *protocol.CommandWriter
	reader *protocol.CommandReader
}

func NewClient(conn net.Conn, index int) *client {
	return &client{
		index:  index,
		conn:   conn,
		reader: protocol.NewCommandReader(conn),
		writer: protocol.NewCommandWriter(conn),
	}
}

func (c *client) Read() (interface{}, error) {
	
	c.conn.SetReadDeadline(time.Now().Add(time.Second * 2))		

	cmd, err := c.reader.Read()

	c.conn.SetReadDeadline(time.Time{})

	return cmd, err
}

func (c *client) WriteCommand(cmd protocol.CommandInterface)  {
	c.writer.WriteCommand(cmd)
}

func (c *client) Close() {
	c.conn.Close()
}