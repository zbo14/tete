package socket

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/zbo14/tete/src/cert"
	"golang.org/x/sys/unix"
	"net"
	"os"
	"time"
)

type Socket struct {
	conn   *tls.Conn
	fd     int
	file   *os.File
	laddr  *net.IPAddr
	lport  int
	myip   net.IP
	peerip net.IP
	raddr  *net.IPAddr
	rport  int
}

func NewSocket(myip net.IP, lport int) (*Socket, error) {
	var domain int
	var sockaddr unix.Sockaddr

	if ipv4 := myip.To4(); ipv4 != nil {
		var addr [4]byte

		domain = unix.AF_INET

		sockaddr = &unix.SockaddrInet4{
			Addr: addr,
			Port: lport,
		}
	} else {
		var addr [16]byte

		domain = unix.AF_INET6

		sockaddr = &unix.SockaddrInet6{
			Addr: addr,
			Port: lport,
		}
	}

	fd, err := unix.Socket(domain, unix.SOCK_STREAM, unix.IPPROTO_TCP)

	if err != nil {
		return nil, err
	}

	var sock Socket

	if err = unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1); err != nil {
		return nil, err
	}

	if err = unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEPORT, 1); err != nil {
		return nil, err
	}

	if err := unix.Bind(fd, sockaddr); err != nil {
		return nil, err
	}

	sock.fd = fd
	sock.file = os.NewFile(uintptr(sock.fd), "tete")
	sock.laddr = &net.IPAddr{IP: myip}
	sock.lport = lport
	sock.myip = myip

	return &sock, nil
}

func (sock *Socket) Close() error {
	return sock.file.Close()
}

func (sock *Socket) Connect(peerip net.IP, port int) error {
	var sockaddr unix.Sockaddr

	if ipv4 := peerip.To4(); ipv4 != nil {
		var addr [4]byte
		copy(addr[:], ipv4)

		sockaddr = &unix.SockaddrInet4{
			Addr: addr,
			Port: port,
		}
	} else {
		ipv6 := peerip.To16()

		var addr [16]byte
		copy(addr[:], ipv6)

		sockaddr = &unix.SockaddrInet6{
			Addr: addr,
			Port: port,
		}
	}

	errs := make(chan error)
	timer := time.NewTimer(3 * time.Second)

	go func() {
		if err := unix.Connect(sock.fd, sockaddr); err != nil {
			errs <- err
			return
		}

		errs <- nil
	}()

	select {
	case err := <-errs:
		timer.Stop()

		sock.peerip = peerip
		sock.raddr = &net.IPAddr{IP: peerip}
		sock.rport = port

		return err

	case <-timer.C:
		return errors.New("socket.Connect() timed out")
	}
}

func (sock *Socket) Listen() error {
	return unix.Listen(sock.fd, unix.SOMAXCONN)
}

func (sock *Socket) Accept(ip net.IP, rport int) (*Socket, error) {
	fd, rsockaddr, err := unix.Accept(sock.fd)

	if err != nil {
		return nil, err
	}

	var actualport int

	if ipv4 := ip.To4(); ipv4 != nil {
		actualip := rsockaddr.(*unix.SockaddrInet4).Addr

		if bytes.Compare(actualip[:], ipv4[:]) != 0 {
			return nil, fmt.Errorf("Expected remote IPv4 address %v, got %v", ipv4, actualip)
		}

		actualport = rsockaddr.(*unix.SockaddrInet4).Port
	} else {
		actualip := rsockaddr.(*unix.SockaddrInet6).Addr
		ipv6 := ip.To16()

		if bytes.Compare(actualip[:], ipv6[:]) != 0 {
			return nil, fmt.Errorf("Expected remote IPv6 address %v, got %v", ipv6, actualip)
		}

		actualport = rsockaddr.(*unix.SockaddrInet6).Port
	}

	if actualport != rport {
		return nil, fmt.Errorf("Expected remote port %d, got %d", rport, actualport)
	}

	lsockaddr, err := unix.Getsockname(fd)

	if err != nil {
		return nil, err
	}

	var laddr *net.IPAddr
	var lport int

	switch lsockaddr.(type) {
	case *unix.SockaddrInet4:
		laddr = &net.IPAddr{IP: lsockaddr.(*unix.SockaddrInet4).Addr[:]}
		lport = lsockaddr.(*unix.SockaddrInet4).Port

	case *unix.SockaddrInet6:
		laddr = &net.IPAddr{IP: lsockaddr.(*unix.SockaddrInet6).Addr[:]}
		lport = lsockaddr.(*unix.SockaddrInet6).Port
	}

	raddr := &net.IPAddr{IP: ip}

	newsock := &Socket{
		fd:    fd,
		file:  os.NewFile(uintptr(fd), "tete"),
		laddr: laddr,
		raddr: raddr,
		lport: lport,
		rport: rport,
	}

	return newsock, nil
}

func (sock *Socket) Secure(isclient bool, keepalive bool) error {
	setKeepAlive := func(clientHello *tls.ClientHelloInfo) (*tls.Config, error) {
		if !keepalive {
			return nil, nil
		}

		if conn, ok := clientHello.Conn.(*net.TCPConn); ok {
			conn.SetKeepAlivePeriod(10 * time.Second)
		}

		return nil, nil
	}

	if isclient {
		config := &tls.Config{
			GetConfigForClient: setKeepAlive,
			InsecureSkipVerify: true,
		}

		sock.conn = tls.Client(sock, config)

		return nil
	}

	pubkey, privkey, err := cert.GenerateKey()

	if err != nil {
		return err
	}

	der, err := cert.CreateCertificate(pubkey, privkey)

	if err != nil {
		return err
	}

	leaf, err := x509.ParseCertificate(der)

	if err != nil {
		return err
	}

	cert := tls.Certificate{
		Certificate: [][]byte{der},
		Leaf:        leaf,
		PrivateKey:  privkey,
	}

	config := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		GetConfigForClient: setKeepAlive,
	}

	sock.conn = tls.Server(sock, config)

	return nil
}

func (sock *Socket) Read(b []byte) (n int, err error) {
	return sock.file.Read(b)
}

func (sock *Socket) Write(b []byte) (n int, err error) {
	return sock.file.Write(b)
}

func (sock *Socket) LocalAddr() net.Addr {
	return sock.laddr
}

func (sock *Socket) RemoteAddr() net.Addr {
	return sock.raddr
}

func (sock *Socket) SetDeadline(t time.Time) error {
	return sock.file.SetDeadline(t)
}

func (sock *Socket) SetReadDeadline(t time.Time) error {
	return sock.file.SetReadDeadline(t)
}

func (sock *Socket) SetWriteDeadline(t time.Time) error {
	return sock.file.SetWriteDeadline(t)
}
