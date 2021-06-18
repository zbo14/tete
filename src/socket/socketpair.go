package socket

import (
	"crypto/tls"
	"errors"
	"log"
	"net"
	"time"
)

type SocketPair struct {
	client   *Socket
	conn     *tls.Conn
	isclient bool
	server   *Socket
}

type Result struct {
	conn     *tls.Conn
	isclient bool
}

func NewSocketPair() (*SocketPair, error) {
	client, err := NewSocket()

	if err != nil {
		return nil, err
	}

	server, err := NewSocket()

	if err != nil {
		return nil, err
	}

	pair := &SocketPair{
		client: client,
		server: server,
	}

	return pair, nil
}

func (pair *SocketPair) Bind(lport int) error {
	if err := pair.client.Bind(lport); err != nil {
		return err
	}

	return pair.server.Bind(lport)
}

func (pair *SocketPair) Connect(ip [4]byte, rport int) error {
	conns := make(chan *Result)
	errs := make(chan error)
	stop := make(chan struct{})

	go func() {
		for i := 0; i < 10; i++ {
			select {
			case <-stop:
				return
			default:
			}

			if err := pair.client.Connect(ip, rport); err != nil {
				log.Println("socket.Connect()", err)
				time.Sleep(time.Second)
				continue
			}

			conns <- &Result{
				conn:     pair.client.conn,
				isclient: true,
			}

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

			conns <- &Result{conn: sock.conn}
			stop <- struct{}{}

			return
		}

		errs <- errors.New("Failed to accept connection from remote peer")
	}()

	select {
	case res := <-conns:
		if err := res.conn.Handshake(); err != nil {
			return err
		}

		pair.conn = res.conn
		pair.isclient = res.isclient

		return nil

	case err := <-errs:
		return err
	}
}

func (pair *SocketPair) IsClient() (bool, error) {
	if pair.conn == nil {
		return false, errors.New("SocketPair not connected")
	}

	return pair.isclient, nil
}

func (pair *SocketPair) Read(b []byte) (n int, err error) {
	if pair.conn == nil {
		return 0, errors.New("SocketPair not connected")
	}

	return pair.conn.Read(b)
}

func (pair *SocketPair) Write(b []byte) (n int, err error) {
	if pair.conn == nil {
		return 0, errors.New("SocketPair not connected")
	}

	return pair.conn.Write(b)
}

func (pair *SocketPair) LocalAddr() net.Addr {
	if pair.isclient {
		return pair.client.laddr
	} else {
		return pair.server.laddr
	}
}

func (pair *SocketPair) RemoteAddr() net.Addr {
	if pair.isclient {
		return pair.client.raddr
	} else {
		return pair.server.raddr
	}
}

func (pair *SocketPair) SetDeadline(t time.Time) error {
	if pair.conn == nil {
		return errors.New("SocketPair not connected")
	}

	return pair.conn.SetDeadline(t)
}

func (pair *SocketPair) SetReadDeadline(t time.Time) error {
	if pair.conn == nil {
		return errors.New("SocketPair not connected")
	}

	return pair.conn.SetReadDeadline(t)
}

func (pair *SocketPair) SetWriteDeadline(t time.Time) error {
	if pair.conn == nil {
		return errors.New("SocketPair not connected")
	}

	return pair.conn.SetWriteDeadline(t)
}
