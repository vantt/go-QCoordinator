package broker

import (
	"sync"
	"context"

	"go.uber.org/zap"
	"github.com/vantt/go-QCoordinator/config"	
	"github.com/vantt/go-QCoordinator/dispatcher"	
	"github.com/vantt/go-QCoordinator/tcpserver"	
)

// Broker ....
type Broker struct {
	config config.BrokerConfig
	logger *zap.Logger	
	server *tcpserver.TcpChatServer
	dispatcher *dispatcher.Dispatcher
	wgChild sync.WaitGroup
}

// NewBroker ...
func NewBroker(cfg config.BrokerConfig, logger *zap.Logger) *Broker{
	return &Broker{config: cfg, logger: logger}
}

// Start ...
func (br *Broker) Start(ctx context.Context, wg *sync.WaitGroup) error {
	if err := br.setup(ctx); err != nil {
		wg.Done()
		return err
	}

	
	// start broker
	go func() {
		defer func() {
			br.wgChild.Wait()
			wg.Done()
			br.logger.Info("Broker QUIT")
		}()

		for {
			select {
			case <-ctx.Done():					
				return
			}
		}
	}()

	br.logger.Info("Broker started")

	return nil
}

func (br *Broker) setup(ctx context.Context) error {
	br.logger.Info("Broker setting up ....")
	
	br.dispatcher = dispatcher.NewDispatcher(br.config, br.logger)
	br.server = tcpserver.NewServer(br.config.Listen, br.dispatcher, br.logger)

	childReady := make(chan string, 3)

	br.wgChild.Add(1)
	if err := br.dispatcher.Start(ctx, &(br.wgChild), childReady); err != nil {
		br.wgChild.Done()
		return err
	}
		
	br.wgChild.Add(1)
	if err := br.server.Start(ctx, &(br.wgChild), childReady); err != nil {
		br.wgChild.Done()
		return err
	}
	
	// wait for sub-goroutine to ready
	for i:=0; i < 2; i++  {
		br.logger.Info(<-childReady)
	}

	close(childReady)

	return nil
}
