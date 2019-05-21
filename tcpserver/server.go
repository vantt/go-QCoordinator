package tcpserver

import (
	"errors"
	"io"
	"net"
	"sync"
	"context"
	"time"
	"fmt"

	"go.uber.org/zap"
	"github.com/vantt/go-QCoordinator/protocol"
	"github.com/vantt/go-QCoordinator/dispatcher"
)

type client struct {
	index  int
	conn   net.Conn	
	writer *protocol.CommandWriter
}

type TcpChatServer struct {
	address string
	listener *net.TCPListener
	clients  []*client
	dispatcher *dispatcher.Dispatcher
	mutex    *sync.Mutex
	logger   *zap.Logger
}

var (
	UnknownClient = errors.New("Unknown client")
)

// NewServer ...
func NewServer(address string, dp *dispatcher.Dispatcher, logger *zap.Logger) *TcpChatServer {
	return &TcpChatServer{
		address: address,
		mutex: &sync.Mutex{},
		dispatcher: dp,
		logger: logger,
	}
}

func (s *TcpChatServer) setup() error {
	var (
		la *net.TCPAddr
		l *net.TCPListener
		err error
	)

	if la, err = net.ResolveTCPAddr("tcp", s.address); err != nil {
		return err
	}


	if l, err = net.ListenTCP("tcp", la); err != nil {
		return err
	}
	
	s.listener = l
	

	s.logger.Info("Tcpserver Listening on " + s.address)

	return nil
}

// Stop ...
func (s *TcpChatServer) Stop() {
	s.listener.Close()
}

// Start ...
func (s *TcpChatServer) Start(ctx context.Context, wg *sync.WaitGroup, readyChan chan<- string ) error {
	if err := s.setup(); err != nil {
		return err
	}

	go func() {
		var wgChild sync.WaitGroup

		defer func() {
			wgChild.Wait()			
			wg.Done()
		}()


		for {
			select {
			case <-ctx.Done():				
				s.Stop()
				return

			default:				
				s.listener.SetDeadline(time.Now().Add(1e9))
				conn, err := s.listener.Accept()

				if err != nil {
					if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
						continue
					}
					s.logger.Error("Failed to accept connection: " +  err.Error())
				}

				// handle connection
				client := s.accept(conn)
				wgChild.Add(1)
				go s.serve(&(wgChild), client)
			}
		}
	}()	

	readyChan <- "TcpServer started."

	return nil
}

func (s *TcpChatServer) Send(client *client, command interface{}) error {	
	return client.writer.WriteCommand(command)
}

func (s *TcpChatServer) accept(conn net.Conn) *client {
	s.logger.Info(fmt.Sprintf("Accepting connection from %v, total clients: %v", conn.RemoteAddr().String(), len(s.clients)+1))

	s.mutex.Lock()
	defer s.mutex.Unlock()

	client := &client{
		index:  len(s.clients),
		conn:   conn,
		writer: protocol.NewCommandWriter(conn),
	}

	s.clients = append(s.clients, client)

	return client
}

func (s *TcpChatServer) remove(client *client) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// remove the connections from clients array
	s.clients = append(s.clients[:client.index], s.clients[client.index+1:]...)

	client.conn.Close()
		
	s.logger.Info(fmt.Sprintf("Closing connection from %v, total clients: %v", client.conn.RemoteAddr().String(), len(s.clients)+1))
}

func (s *TcpChatServer) serve(wg *sync.WaitGroup, client *client) {
	defer func() {
		s.remove(client)
		wg.Done()
	}()

	s.Send(client, &protocol.HelloCommand{})

	cmdReader := protocol.NewCommandReader(client.conn)

	for {
		cmd, err := cmdReader.Read()
		
		if err != nil && err != io.EOF {
			s.logger.Error(fmt.Sprintf("Read error: %v", err))
		}

		if cmd != nil {
			switch v := cmd.(type) {
			case *protocol.ReserveCommand:				
				go func() {
					select {
					case task, ok := <-s.dispatcher.Reserve():
						if ok {
							cmd := &protocol.TaskCommand{
								ID:  task.ID, 
								Queue: task.QueueName, 
								Payload: task.Payload.([]byte),
							}
		
							s.Send(client, cmd)
						} else {
							s.Send(client, &protocol.NoTaskCommand{} )
						}
					default:
						s.Send(client, &protocol.NoTaskCommand{} )
					}					
				}()

			case *protocol.ResultCommand:				
				go func() {
					result := &dispatcher.TaskResult{ 
						ID: v.ID, 
						ExitCode: v.ExitCode,
					}

					s.dispatcher.Done(result)
				}()
			}
		}

		if err == io.EOF {
			break
		}
	}
}
