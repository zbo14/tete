package socket

import (
	"fmt"
	"golang.org/x/sys/unix"
)

type Addr struct {
	ip       [4]byte
	port     int
	sockaddr unix.Sockaddr
}

func (addr *Addr) Network() string {
	return "tcp"
}

func (addr *Addr) String() string {
	return fmt.Sprintf(
		"%d.%d.%d.%d:%d",
		addr.ip[0],
		addr.ip[1],
		addr.ip[2],
		addr.ip[3],
		addr.port,
	)
}
