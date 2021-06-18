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
)

func main() {
	var ipstr string
	var lport int
	var rport int
	var verbose bool

	flag.StringVar(&ipstr, "ip", "", "public IPv4 address to connect to")
	flag.IntVar(&lport, "lport", 54312, "local port to listen on")
	flag.IntVar(&rport, "rport", 54312, "remote port to connect to")
	flag.BoolVar(&verbose, "v", false, "increase logging verbosity")

	flag.Parse()

	ip := net.ParseIP(ipstr)

	if ip == nil {
		log.Fatalln(errors.New("Invalid IPv4 address"))
	}

	if lport < 0 || lport > 65535 {
		log.Fatalln(errors.New("Local port must be >= 0 and < 65536"))
	}

	if rport < 0 || rport > 65535 {
		log.Fatalln(errors.New("Remote port must be >= 0 and < 65536"))
	}

	if !verbose {
		log.SetOutput(ioutil.Discard)
	}

	pair, err := socket.NewSocketPair()

	if err != nil {
		log.Fatalln(err)
	}

	if err := pair.Bind(lport); err != nil {
		log.Fatalln(err)
	}

	var ipbuf [4]byte
	copy(ipbuf[:], ip[len(ip)-4:])

	if err := pair.Connect(ipbuf, rport); err != nil {
		log.Fatalln(err)
	}

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

	log.Println("Peer closed connection")
}
