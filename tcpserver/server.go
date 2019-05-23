package tcpserver

import (	
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

type TcpServer struct {
	address string
	listener *net.TCPListener
	clients  []*client
	dispatcher *dispatcher.Dispatcher
	mutex    *sync.Mutex
	logger   *zap.Logger
}

// NewServer ...
func NewServer(address string, dp *dispatcher.Dispatcher, logger *zap.Logger) *TcpServer {
	return &TcpServer{
		address: address,
		mutex: &sync.Mutex{},
		dispatcher: dp,
		logger: logger,
	}
}

func (s *TcpServer) setup() error {
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
func (s *TcpServer) Stop() {
	s.listener.Close()
}

// Start ...
func (s *TcpServer) Start(ctx context.Context, wg *sync.WaitGroup, readyChan chan<- string ) error {
	if err := s.setup(); err != nil {
		return err
	}

	go func() {
		var wgChild sync.WaitGroup

		defer func() {
			wgChild.Wait()			
			wg.Done()
			s.logger.Info("Tcpserver QUIT.")
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
				go s.serve(ctx, &(wgChild), client)
			}
		}
	}()	

	readyChan <- "TcpServer started."

	return nil
}

func (s *TcpServer) SendCommand(client *client, cmd protocol.CommandInterface) {	
	client.WriteCommand(cmd)
}

func (s *TcpServer) accept(conn net.Conn) *client {
	s.logger.Info(fmt.Sprintf("Accepting connection from %v, total clients: %v", conn.RemoteAddr().String(), len(s.clients)+1))

	s.mutex.Lock()
	defer s.mutex.Unlock()

	client := NewClient(conn, len(s.clients))
		
	s.clients = append(s.clients, client)

	return client
}

func (s *TcpServer) remove(client *client) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	client.Close()

	// remove the connections from clients array
	s.clients = append(s.clients[:client.index], s.clients[client.index+1:]...)
	
	s.logger.Info(fmt.Sprintf("Closing connection from %v, total clients: %v", client.conn.RemoteAddr().String(), len(s.clients)+1))
}

func (s *TcpServer) serve(ctx context.Context, wg *sync.WaitGroup, client *client) {
	var wgChild sync.WaitGroup

	defer func() {
		wgChild.Wait()
		s.remove(client)
		wg.Done()
	}()

	client.WriteCommand(&protocol.HelloCommand{})

	for {
		select {
			case <-ctx.Done():				
				return

			default:					
				cmd, err := client.Read()

				if err == nil && cmd != nil {
					switch v := cmd.(type) {					
					case *protocol.ReserveCommand:
						wgChild.Add(1)		
						go s.handleReserveCommand(ctx, &(wgChild), client)

					case *protocol.ResultCommand:				
						wgChild.Add(1)	
						go s.handleResultCommand(&(wgChild), v)
					}
				}

				if (err == protocol.UnknownCommandErr) {
					client.WriteCommand(&protocol.UnknownCommand{})
				} else if err == io.EOF {
					return
				} 
		}
	}
}

func (s *TcpServer) handleReserveCommand(ctx context.Context, wg *sync.WaitGroup, client *client) {

	tickerTimeOut := time.NewTicker(1 * time.Second)

	defer func() {
	 	tickerTimeOut.Stop()
		wg.Done()
	}()

	for {
		select {
		case <-ctx.Done():			
			client.WriteCommand(&protocol.NoTaskCommand{})
			return

		case <-tickerTimeOut.C:
			client.WriteCommand(&protocol.NoTaskCommand{})
			return

		case task, ok := <-s.dispatcher.Reserve():
			if ok {
				cmd := &protocol.TaskCommand{
					ID:  task.ID, 
					Queue: task.QueueName, 
					Payload: task.Payload.([]byte),
				}

				client.WriteCommand(cmd)
				
				return
			}
		}
	}
}

func (s *TcpServer) handleResultCommand(wg *sync.WaitGroup, cmd *protocol.ResultCommand) {
	defer wg.Done()

	s.dispatcher.Done(dispatcher.NewTaskResult(cmd.ID, cmd.ExitCode))
}