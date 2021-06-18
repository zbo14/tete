package socket

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/zbo14/tete/src/cert"
	"golang.org/x/sys/unix"
	"net"
	"os"
	"time"
)

type Socket struct {
	conn  *tls.Conn
	fd    int
	file  *os.File
	laddr *Addr
	raddr *Addr
}

func NewSocket() (*Socket, error) {
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM, unix.IPPROTO_TCP)

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

	sock.fd = fd

	return &sock, nil
}

func (sock *Socket) Bind(lport int) error {
	ip := [4]byte{0, 0, 0, 0}

	sockaddr := &unix.SockaddrInet4{
		Port: lport,
		Addr: ip,
	}

	if err := unix.Bind(sock.fd, sockaddr); err != nil {
		return err
	}

	sock.file = os.NewFile(uintptr(sock.fd), "tete")

	sock.laddr = &Addr{
		ip:       ip,
		port:     lport,
		sockaddr: sockaddr,
	}

	return nil
}

func (sock *Socket) Close() error {
	return sock.file.Close()
}

func (sock *Socket) Connect(ip [4]byte, rport int) error {
	sockaddr := &unix.SockaddrInet4{
		Port: rport,
		Addr: ip,
	}

	if err := unix.Connect(sock.fd, sockaddr); err != nil {
		return err
	}

	sock.raddr = &Addr{
		ip:       ip,
		port:     rport,
		sockaddr: sockaddr,
	}

	sock.conn = tls.Client(sock, &tls.Config{InsecureSkipVerify: true})

	return nil
}

func (sock *Socket) Listen() error {
	return unix.Listen(sock.fd, unix.SOMAXCONN)
}

func (sock *Socket) Accept(ip [4]byte, rport int) (*Socket, error) {
	fd, rsockaddr, err := unix.Accept(sock.fd)

	if err != nil {
		return nil, err
	}

	actualip := rsockaddr.(*unix.SockaddrInet4).Addr

	if bytes.Compare(actualip[:], ip[:]) != 0 {
		return nil, fmt.Errorf("Expected remote IPv4 address %v, got %v", ip, actualip)
	}

	actualport := rsockaddr.(*unix.SockaddrInet4).Port

	if actualport != rport {
		return nil, fmt.Errorf("Expected remote port %d, got %d", rport, actualport)
	}

	lsockaddr, err := unix.Getsockname(fd)

	if err != nil {
		return nil, err
	}

	pubkey, privkey, err := cert.GenerateKey()

	if err != nil {
		return nil, err
	}

	der, err := cert.CreateCertificate(pubkey, privkey)

	if err != nil {
		return nil, err
	}

	leaf, err := x509.ParseCertificate(der)

	if err != nil {
		return nil, err
	}

	laddr := &Addr{
		ip:       lsockaddr.(*unix.SockaddrInet4).Addr,
		port:     lsockaddr.(*unix.SockaddrInet4).Port,
		sockaddr: lsockaddr,
	}

	raddr := &Addr{
		ip:       ip,
		port:     rport,
		sockaddr: rsockaddr,
	}

	newsock := &Socket{
		fd:    fd,
		file:  os.NewFile(uintptr(fd), "tete"),
		laddr: laddr,
		raddr: raddr,
	}

	cert := tls.Certificate{
		Certificate: [][]byte{der},
		Leaf:        leaf,
		PrivateKey:  privkey,
	}

	newsock.conn = tls.Server(newsock, &tls.Config{Certificates: []tls.Certificate{cert}})

	return newsock, nil
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
