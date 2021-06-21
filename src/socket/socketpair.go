package socket

import (
	"bytes"
	"errors"
	"log"
	"net"
	"time"
)

type SocketPair struct {
	client      *Socket
	server      *Socket
	isconnected bool
}

func NewSocketPair(myip net.IP, lport int) (*SocketPair, error) {
	client, err := NewSocket(myip, lport)

	if err != nil {
		return nil, err
	}

	server, err := NewSocket(myip, lport)

	if err != nil {
		return nil, err
	}

	pair := &SocketPair{
		client: client,
		server: server,
	}

	return pair, nil
}

func Connect(myip net.IP, lport int, peerip net.IP, rport int) (*SocketPair, error) {
	pair, err := NewSocketPair(myip, lport)

	if err != nil {
		return nil, err
	}

	isclient := bytes.Compare(myip, peerip) == 1

	if err := pair.Connect(peerip, rport, isclient); err != nil {
		return nil, err
	}

	return pair, nil
}

func (pair *SocketPair) Close() error {
	if pair.client != nil {
		if err := pair.client.Close(); err != nil {
			return err
		}
	}

	if err := pair.server.Close(); err != nil {
		return err
	}

	return nil
}

func (pair *SocketPair) Connect(peerip net.IP, rport int, isclient bool) error {
	socks := make(chan *Socket)
	errs := make(chan error)
	stop := make(chan struct{})

	go func() {
		for i := 0; i < 10; i++ {
			select {
			case <-stop:
				return

			default:
				if err := pair.client.Connect(peerip, rport); err != nil {
					log.Println("socket.Connect()", err)
					time.Sleep(time.Second)
					continue
				}

				socks <- pair.client

				return
			}
		}

		errs <- errors.New("Failed to connect to remote peer")
	}()

	go func() {
		if err := pair.server.Listen(); err != nil {
			errs <- err
			return
		}

		for i := 0; i < 10; i++ {
			sock, err := pair.server.Accept(peerip, rport)

			if err != nil {
				log.Println("socket.Accept()", err)
				continue
			}

			socks <- sock
			stop <- struct{}{}

			return
		}

		errs <- errors.New("Failed to accept connection from remote peer")
	}()

	select {
	case sock := <-socks:
		if err := sock.Secure(isclient); err != nil {
			return err
		}

		pair.client = sock
		pair.isconnected = true

		return nil

	case err := <-errs:
		return err
	}
}

func (pair *SocketPair) Read(b []byte) (n int, err error) {
	if !pair.isconnected {
		return 0, errors.New("SocketPair not connected")
	}

	return pair.client.conn.Read(b)
}

func (pair *SocketPair) Write(b []byte) (n int, err error) {
	if !pair.isconnected {
		return 0, errors.New("SocketPair not connected")
	}

	return pair.client.conn.Write(b)
}

func (pair *SocketPair) LocalAddr() net.Addr {
	return pair.client.laddr
}

func (pair *SocketPair) RemoteAddr() net.Addr {
	return pair.client.raddr
}

func (pair *SocketPair) SetDeadline(t time.Time) error {
	if !pair.isconnected {
		return errors.New("SocketPair not connected")
	}

	return pair.client.conn.SetDeadline(t)
}

func (pair *SocketPair) SetReadDeadline(t time.Time) error {
	if !pair.isconnected {
		return errors.New("SocketPair not connected")
	}

	return pair.client.conn.SetReadDeadline(t)
}

func (pair *SocketPair) SetWriteDeadline(t time.Time) error {
	if !pair.isconnected {
		return errors.New("SocketPair not connected")
	}

	return pair.client.conn.SetWriteDeadline(t)
}
