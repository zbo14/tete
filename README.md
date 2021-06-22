# tete

As in *tête-à-tête*.

Peer-to-peer TLS connections through firewalls via TCP hole-punching.

## Why?

The goal: easily allow 2 peers to establish a secure connection after exchanging public IP addresses and port numbers.

However, there are several difficulties with creating and maintaining encrypted p2p connections:

* NAT routers blocking/dropping TCP connections between peers on different networks
* Coordinating TLS negotiation when neither peer is hosting a persistent server on a domain
* Adapting to socket APIs that are based on a client-server paradigm

`tete` addresses these challenges and creates abstractions that hopefully make secure p2p communication easier for people and other applications.

This repository contains a command line tool that does the following:

* Establish TLS connections between peers on different networks
* Read data from stdin and send it over a connection
* Read encrypted data from connection and write decrypted data to stdout

This makes `tete` easy to use with other tools (e.g. by piping to UNIX commands or other CLIs).

## Install

`go get -u github.com/zbo14/tete`

## Usage

Suppose Alice has public IPv4 address, "1.2.3.4", and port number, 12345.

Bob has public IPv4 address, "5.6.7.8", and port number, 56789.

In her terminal, Alice would type:

`$ tete -myip 1.2.3.4 -peerip 5.6.7.8 -lport 12345 -rport 56789`

In his terminal, Bob would type:

`$ tete -myip 5.6.7.8 -peerip 1.2.3.4 -lport 56789 -rport 12345`

**Note:** each peer *must* specify their own public IP address. IP address comparison determines which peer acts as a server (and which acts as a client) in the TLS handshake.

After pressing enter, they both should see the following printed to stderr:

`<date> <time> Connected to peer!`

Now, they can write to stdin to send data over the secure connection and read data from stdout.

The connection closes when Alice or Bob issues an interrupt signal.

They should see the following printed to stderr:

`<date> <time> Closed connection` or `<date> <time> Peer closed connection`

```
Usage of tete:
    -h  show usage information and exit
    -k  enable TCP keepalives
    -lport int
        local port you're listening on (default 54312)
    -myip string
        your public IPv4/IPv6 address
    -peerip string
        peer's public IPv4/IPv6 address
    -rport int
        remote port the peer's listening on (default 54312)
    -v  increases logging verbosity
```
