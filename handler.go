package main

import (
	"context"
	"encoding/binary"
	"log"
	"net"
	"time"
)

type Handler interface {
	Handle(ctx context.Context, tcpConn *net.TCPConn)
}

type HandleFunc func(ctx context.Context, tcpConn *net.TCPConn)

func (hf HandleFunc) Handle(ctx context.Context, tcpConn *net.TCPConn) {
	hf(ctx, tcpConn)
}

func CloseHandler(handler Handler) Handler {
	return HandleFunc(func(ctx context.Context, tcpConn *net.TCPConn) {
		defer func() {
			if err := tcpConn.Close(); err != nil {
				rAddr := tcpConn.RemoteAddr()
				log.Printf("failed to close connection, remote address=%s, because of: %v\n", rAddr, err)
			}
		}()
		handler.Handle(ctx, tcpConn)
	})
}

func PanicRecoverHandler(handler Handler) Handler {
	return HandleFunc(func(ctx context.Context, tcpConn *net.TCPConn) {
		defer func() {
			if cause := recover(); cause != nil {
				log.Printf("server process panic occured, because of: %v\n", cause)
			}
		}()
		handler.Handle(ctx, tcpConn)
	})
}

func LoggingHandler(handler Handler) Handler {
	return HandleFunc(func(ctx context.Context, tcpConn *net.TCPConn) {
		rAddr := tcpConn.RemoteAddr()
		defer func() {
			log.Printf("end serve connection, remote address=%s\n", rAddr)
		}()
		log.Printf("start serve connection, remote address=%s\n", rAddr)
		handler.Handle(ctx, tcpConn)
	})
}

type ConfigurationOption func(ctx context.Context, tcpConn *net.TCPConn) (fatal bool, err error)

func DisableKeepAliveOption(ctx context.Context, tcpConn *net.TCPConn) (fatal bool, err error) {
	return false, tcpConn.SetKeepAlive(false)
}

func DisableDelayOption(ctx context.Context, tcpConn *net.TCPConn) (fatal bool, err error) {
	return false, tcpConn.SetNoDelay(true)
}

func ConfigurationHandler(handler Handler, confOpts ...ConfigurationOption) Handler {
	return HandleFunc(func(ctx context.Context, tcpConn *net.TCPConn) {
		for i, confOpt := range confOpts {
			if fatal, err := confOpt(ctx, tcpConn); err != nil {
				rAddr := tcpConn.RemoteAddr()
				log.Printf("configuration failed, remote address=%s, option=%d, because of: %v\n", rAddr, i+1, err)
				if fatal {
					log.Printf("configuration error is fatal, futher handling terminated, remote address=%s\n", rAddr)
					return
				}
			}
		}
		handler.Handle(ctx, tcpConn)
	})
}

type UnixTimeHandler struct {
	From time.Time
}

func (uth UnixTimeHandler) Handle(ctx context.Context, tcpConn *net.TCPConn) {
	secs := uint32(time.Now().UTC().Sub(uth.From).Seconds())
	if err := binary.Write(tcpConn, binary.BigEndian, secs); err != nil {
		rAddr := tcpConn.RemoteAddr()
		log.Printf("failed to send response, remote address=%s, because of: %v\n", rAddr, err)
		return
	}
}
