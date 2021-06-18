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
	"strings"
)

func main() {
	var myipstr string
	var peeripstr string
	var lport int
	var rport int
	var verbose bool

	flag.StringVar(&myipstr, "myip", "", "your public IPv4 address")
	flag.StringVar(&peeripstr, "peerip", "", "peer's public IPv4 address")
	flag.IntVar(&lport, "lport", 54312, "local port you're listening on")
	flag.IntVar(&rport, "rport", 54312, "remote port the peer's listening on")
	flag.BoolVar(&verbose, "v", false, "increases logging verbosity")

	flag.Parse()

	myip := net.ParseIP(myipstr)

	if myip == nil {
		log.Fatalln(errors.New("Invalid public IPv4 address"))
	}

	peerip := net.ParseIP(peeripstr)

	if peerip == nil {
		log.Fatalln(errors.New("Invalid IPv4 address for peer"))
	}

	ipcmp := strings.Compare(myipstr, peeripstr)

	if ipcmp == 0 {
		log.Fatalln(errors.New("You and peer cannot use same IPv4 address"))
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

	pair, err := socket.NewSocketPair()

	if err != nil {
		log.Fatalln(err)
	}

	if err := pair.Bind(lport); err != nil {
		log.Fatalln(err)
	}

	var ipbuf [4]byte

	copy(ipbuf[:], peerip[len(peerip)-4:])

	isclient := ipcmp == 1

	if err := pair.Connect(ipbuf, rport, isclient); err != nil {
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
