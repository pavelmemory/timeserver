package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

const (
	InvalidConfiguration = 2
	StartFailed          = 3
)

func main() {
	conf, err := initConfiguration()
	if err != nil {
		log.Println(err)
		os.Exit(InvalidConfiguration)
	}

	listener, err := initListener(conf.Port)
	if err != nil {
		log.Println(err)
		os.Exit(StartFailed)
	}

	timeSrv := NewServer(
		listener,
		CloseHandler(
			PanicRecoverHandler(
				LoggingHandler(
					ConfigurationHandler(
						UnixTimeHandler{From: January1Of1900},
						DisableKeepAliveOption,
						DisableDelayOption,
					),
				),
			),
		),
	)

	ctx, ctxCancel := context.WithCancel(context.Background())

	// listening for OS signals to stop
	go func() {
		defer ctxCancel()
		<-ListenSignals(os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM)

		if err := timeSrv.Shutdown(); err != nil {
			log.Printf("server stopped, port=%d, because of: %v\n", conf.Port, err)
		} else {
			log.Printf("server stopped, port=%d\n", conf.Port)
		}
	}()

	log.Printf("server started, port=%d\n", conf.Port)
	if err := timeSrv.ListenAndServe(ctx); err != nil {
		log.Printf("server stopped, port=%d, because of: %v\n", conf.Port, err)
	}
	<-ctx.Done()
}

func ListenSignals(signals ...os.Signal) chan os.Signal {
	sigListener := make(chan os.Signal)
	signal.Notify(sigListener, signals...)
	return sigListener
}
