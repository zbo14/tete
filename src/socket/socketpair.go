package socket

import (
	"crypto/tls"
	"errors"
	"log"
	"net"
	"time"
)

type SocketPair struct {
	client *Socket
	server *Socket
	sock   *Socket
}

type Result struct {
	conn     *tls.Conn
	isclient bool
}

func NewSocketPair() (*SocketPair, error) {
	server, err := NewSocket()

	if err != nil {
		return nil, err
	}

	pair := &SocketPair{server: server}

	return pair, nil
}

func (pair *SocketPair) Bind(lport int) error {
	return pair.server.Bind(lport)
}

func (pair *SocketPair) Connect(ip [4]byte, rport int, isclient bool) error {
	socks := make(chan *Socket)
	errs := make(chan error)
	stop := make(chan struct{})

	go func() {
		for i := 0; i < 10; i++ {
			select {
			case <-stop:
				return
			default:
			}

			client, err := NewSocket()

			if err != nil {
				errs <- err
				client.Close()
				return
			}

			if err := client.Bind(pair.server.laddr.port); err != nil {
				errs <- err
				client.Close()
				return
			}

			if err := client.Connect(ip, rport); err != nil {
				client.Close()
				log.Println("socket.Connect()", err)
				time.Sleep(time.Second)
				continue
			}

			pair.client = client
			socks <- pair.client

			return
		}

		errs <- errors.New("Failed to connect to remote peer")
	}()

	go func() {
		if err := pair.server.Listen(); err != nil {
			errs <- err
			return
		}

		for i := 0; i < 10; i++ {
			sock, err := pair.server.Accept(ip, rport)

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
		pair.server.Close()

		if err := sock.Secure(isclient); err != nil {
			return err
		}

		pair.sock = sock

		return nil

	case err := <-errs:
		return err
	}
}

func (pair *SocketPair) Read(b []byte) (n int, err error) {
	if pair.sock == nil {
		return 0, errors.New("SocketPair not connected")
	}

	return pair.sock.conn.Read(b)
}

func (pair *SocketPair) Write(b []byte) (n int, err error) {
	if pair.sock == nil {
		return 0, errors.New("SocketPair not connected")
	}

	return pair.sock.conn.Write(b)
}

func (pair *SocketPair) LocalAddr() net.Addr {
	return pair.sock.laddr
}

func (pair *SocketPair) RemoteAddr() net.Addr {
	return pair.sock.raddr
}

func (pair *SocketPair) SetDeadline(t time.Time) error {
	if pair.sock == nil {
		return errors.New("SocketPair not connected")
	}

	return pair.sock.conn.SetDeadline(t)
}

func (pair *SocketPair) SetReadDeadline(t time.Time) error {
	if pair.sock == nil {
		return errors.New("SocketPair not connected")
	}

	return pair.sock.conn.SetReadDeadline(t)
}

func (pair *SocketPair) SetWriteDeadline(t time.Time) error {
	if pair.sock == nil {
		return errors.New("SocketPair not connected")
	}

	return pair.sock.conn.SetWriteDeadline(t)
}
