package main

import (
	"context"
	"log"
	"net"
	"strings"
	"sync"
)

// TimeServerCtxKey value can be used in the Handler method execution to get access to the server
const TimeServerCtxKey = 0xDEADBEEF

type TCPServer interface {
	// Listens for incoming requests and properly handles them
	// Returns error received from `AcceptTCP` of underlying listener
	ListenAndServe(ctx context.Context) error
	// Stops from listening for incoming requests
	// Blocks until all accepted requests are not served
	// Returns error received from `Close` of underlying listener
	Shutdown() error
}

// NewServer return new TCPServer and requires all parameters to be not `nil` values
func NewServer(listener *net.TCPListener, handler Handler) *tcpServer {
	return &tcpServer{
		listener: listener,
		handler:  handler,
	}
}

type tcpServer struct {
	wg       sync.WaitGroup
	closer   sync.Once
	closeErr error
	listener *net.TCPListener
	ctx      context.Context
	stop     context.CancelFunc
	handler  Handler
}

func (ts *tcpServer) Shutdown() error {
	ts.closer.Do(func() {
		if ts.stop != nil {
			ts.stop()
		}
		ts.closeErr = ts.listener.Close()
		ts.wg.Wait() // wait until all accepted connections would be served
	})
	return ts.closeErr
}

// ListenAndServe starts server to accept incoming requests and handle them with provided handler
// Method returns if passed ctx was cancelled/timed out and closes provided listener
func (ts *tcpServer) ListenAndServe(ctx context.Context) error {
	ts.ctx, ts.stop = context.WithCancel(ctx)
	defer ts.stop()

	incoming := make(chan *net.TCPConn, 1024) // magic number - needs to be configured
	defer close(incoming)

	go func() {
		// we must serve all accepted connections before termination
		for tcpConn := range incoming {
			// each in it's own goroutine - pool of workers as future optimisation
			ts.wg.Add(1)
			go func() {
				defer ts.wg.Done()
				// each running handler have access to the server by which it is served
				hCtx := context.WithValue(ts.ctx, TimeServerCtxKey, ts)
				ts.handler.Handle(hCtx, tcpConn)
			}()
		}
	}()

	for {
		// loop exists when someone closes listener
		tcpConn, err := ts.listener.AcceptTCP()
		if err != nil {
			if netOpErr, ok := err.(*net.OpError); ok && netOpErr.Temporary() {
				log.Printf("temporary network error because of: %v\n", err.Error())
			}

			// global var with this error located in the internal package, so workaround as stdlib has
			if strings.Contains(err.Error(), "use of closed network connection") {
				return nil
			}

			log.Printf("abort listening because of: %v\n", err.Error())
			ts.Shutdown() // cancels internal context that leads to closing of the listener
			return err
		}

		incoming <- tcpConn
	}
}
