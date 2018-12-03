package main

import (
	"context"
	"encoding/binary"
	"net"
	"sync"
	"testing"
	"time"
)

// Can be converted to the cli client, but unfortunately I have no time for this.
// Same as for the other tests that needs to be written for Handler(s) and so on...
// Sorry for that.
// It's not because I am lazy, it's just because I have hard times now (moving to the other city)
// Any way I would appreciate if you provide a feed back about the done work, thanks!
func getTime(t *testing.T, address string) uint32 {
	t.Helper()
	addr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		t.Fatal(err)
	}

	tcpConn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		t.Fatal(err)
	}
	defer tcpConn.Close()

	if err := tcpConn.SetKeepAlive(false); err != nil {
		t.Fatal(err)
	}

	if err := tcpConn.CloseWrite(); err != nil {
		t.Fatal(err)
	}

	var result uint32
	if err := binary.Read(tcpConn, binary.BigEndian, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func setupTimeServer(t *testing.T, addr *net.TCPAddr, withTime time.Time) (TCPServer, net.Addr) {
	t.Helper()
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	return NewServer(
		listener,
		CloseHandler(
			ConfigurationHandler(UnixTimeHandler{From: withTime}, DisableKeepAliveOption, DisableDelayOption)),
	), listener.Addr()
}

func TestTimeNistGov(t *testing.T) {
	t.Parallel()
	srv, addr := setupTimeServer(t, nil, January1Of1900)
	go srv.ListenAndServe(context.Background())
	defer srv.Shutdown()
	time.Sleep(time.Second)

	expected := getTime(t, "time.nist.gov:37")
	actual := getTime(t, addr.String())

	var diff uint32
	if expected > actual {
		diff = expected - actual
	} else {
		diff = actual - expected
	}

	if diff > 5 {
		t.Fatalf("actual time %d differs from expected %d too significant: %d", actual, expected, diff)
	}
}

func TestTimeMulti(t *testing.T) {
	t.Parallel()
	srv, addr := setupTimeServer(t, nil, January1Of1900)
	go srv.ListenAndServe(context.Background())
	defer srv.Shutdown()
	time.Sleep(time.Second)

	expected := getTime(t, "time.nist.gov:37")

	var wg sync.WaitGroup
	defer wg.Wait()
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			actual := getTime(t, addr.String())

			var diff uint32
			if expected > actual {
				diff = expected - actual
			} else {
				diff = actual - expected
			}

			if diff > 5 {
				t.Fatalf("actual time %d differs from expected %d too significant: %d", actual, expected, diff)
			}
		}()
	}

	timer := time.AfterFunc(5*time.Second, func() {
		t.Fatal("test runs too long...")
	})
	defer timer.Stop()
}
