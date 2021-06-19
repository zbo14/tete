package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"github.com/zbo14/tete/src/socket"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	var help bool
	var myipaddr string
	var peeripaddr string
	var lport int
	var rport int
	var verbose bool

	flag.BoolVar(&help, "h", false, "show usage information and exit")
	flag.StringVar(&myipaddr, "myip", "", "your public IPv4/IPv6 address")
	flag.StringVar(&peeripaddr, "peerip", "", "peer's public IPv4/IPv6 address")
	flag.IntVar(&lport, "lport", 54312, "local port you're listening on")
	flag.IntVar(&rport, "rport", 54312, "remote port the peer's listening on")
	flag.BoolVar(&verbose, "v", false, "increases logging verbosity")

	flag.Parse()

	if help {
		fmt.Fprintln(os.Stderr, `Usage of tete:
	-h	show usage information and exit
  	-lport int
    	local port you're listening on (default 54312)
  	-myip string
    	your public IPv4/IPv6 address
  	-peerip string
    	peer's public IPv4/IPv6 address
  	-rport int
    	remote port the peer's listening on (default 54312)
  	-v	increases logging verbosity`)

		os.Exit(0)
	}

	myip := net.ParseIP(myipaddr)

	if myip == nil {
		log.Fatalln(errors.New("Invalid public IP address"))
	}

	peerip := net.ParseIP(peeripaddr)

	if peerip == nil {
		log.Fatalln(errors.New("Invalid IP address for peer"))
	}

	if myip.Equal(peerip) {
		log.Fatalln(errors.New("Cannot have same IP address as peer"))
	}

	if lport < 1 || lport > 65535 {
		log.Fatalln(errors.New("Local port must be > 0 and < 65536"))
	}

	if rport < 1 || rport > 65535 {
		log.Fatalln(errors.New("Remote port must be > 0 and < 65536"))
	}

	if !verbose {
		log.SetOutput(ioutil.Discard)
	}

	pair, err := socket.Connect(myip, lport, peerip, rport)

	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Connected to peer!")

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-ch
		pair.Close()
		log.Println("Closed connection")
		os.Exit(0)
	}()

	go func() {
		var buf bytes.Buffer
		scanner := bufio.NewScanner(os.Stdin)

		for scanner.Scan() {
			text := scanner.Text()

			buf.WriteString(text)
			buf.WriteString("\n")

			n, err := pair.Write(buf.Bytes())

			if err != nil {
				log.Fatalln(err)
			}

			if buf.Len() != n {
				log.Fatalln(errors.New("Failed to write entire message"))
			}

			buf.Reset()
		}
	}()

	scanner := bufio.NewScanner(pair)

	for scanner.Scan() {
		text := scanner.Text()
		fmt.Printf("Message from peer: %s\n", text)
	}

	pair.Close()
	log.Println("Peer closed connection")
}