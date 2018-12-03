package main

import (
	"errors"
	"flag"
	"net"
	"time"
)

var January1Of1900 = time.Date(1900, time.January, 1, 0, 0, 0, 0, time.UTC)

type Configuration struct {
	Port int
}

func (c *Configuration) Initialize() error {
	portPtr := flag.String("p", "", "port number to listen (required)")
	flag.Parse()

	if *portPtr == "" {
		flag.PrintDefaults()
		return errors.New("required flag '-p' was not provided")
	}

	port, err := net.LookupPort("tcp", *portPtr)
	if err != nil {
		return err
	}

	c.Port = port
	return nil
}
