package tcpserver

import (
	"sync"
	"net"	
	"time"	
	"github.com/vantt/go-QCoordinator/protocol"
	"math/rand"	
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

//////////////////////////

type clientList struct {
	clients map[int]*client
	sync.RWMutex
}

func NewClientList() *clientList {
	return &clientList{clients: make(map[int]*client)}
}

func (l *clientList) Add(conn net.Conn) *client {
	
	var index int

	for {
		index = rand.Int()

		if _, exist := l.clients[index]; !exist {
			l.Lock();
			defer l.Unlock();
			break
		}
	}
		
	client := NewClient(conn, index)

	l.clients[index] = client

	return client
}

func (l *clientList) Remove(client *client) {
	l.Lock()
	defer l.Unlock()

	client.Close()

	// remove the connections from clients array
	delete(l.clients, client.index)	
}

func (l *clientList) Len() int {
	return len(l.clients)	
}

func (l *clientList) Cleanup() {
	for _, client := range l.clients { 
		l.Remove(client)
	}
}