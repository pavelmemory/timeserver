package main

import (
	"net"
	"strconv"
)

func initConfiguration() (Configuration, error) {
	conf := Configuration{}
	return conf, conf.Initialize()
}

func initListener(port int) (*net.TCPListener, error) {
	addr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort("0.0.0.0", strconv.Itoa(port)))
	if err != nil {
		return nil, err
	}

	return net.ListenTCP("tcp", addr)
}
