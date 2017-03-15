package main

import (
	"errors"
	"flag"
	"log"
	"os"
	"time"
)

type Remote interface {
	Channel() chan []byte
	Recv(timeout time.Duration) ([]byte, error)
	Write(buf []byte, timeout time.Duration) (int, error)
	Open() chan []byte
	Close() error
}

type Flags struct {
	help      bool
	stdio     bool
	list      bool
	remote    string
	tls       bool
	reconnect bool
}

func parseFlags() Flags {
	flags := new(Flags)

	helpFlag := flag.Bool("help", false, "Show help text")
	stdioFlag := flag.Bool("stdio", false, "Use stdio for communication instead of remote")
	listFlag := flag.Bool("list", false, "List available serialports")
	remoteFlag := flag.String("remote", "tcp.cloud.tiny-mesh.com:7002", "The upstream url to connect to")
	usetlsFlag := flag.Bool("tls", true, "Controll use of TLS with --remote")
	reconnectFlag := flag.Bool("reconnect", false, "Automatically reconnect upstream (tcp,tls) on failure")

	flag.Parse()

	flags.help = *helpFlag
	flags.stdio = *stdioFlag
	flags.list = *listFlag
	flags.remote = *remoteFlag
	flags.tls = *usetlsFlag
	flags.reconnect = *reconnectFlag

	return *flags
}

func main() {

	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	flags := parseFlags()

	if true == flags.help {
		flag.PrintDefaults()
		return
	} else if true == flags.list {
		PrintPortList()
		return
	}

	path := flag.Arg(0)

	if "" == path {
		log.Fatal(errors.New("1st argument, tty path, missing"))
	}

	var upstream Remote
	var downstream Remote
	var err error

	log.Printf("serial: opening %v\n", path)
	downstream, err = ConnectSerial(path)

	if nil != err {
		log.Fatal(err)
	}

	connectUpstream := func() (Remote, error) {

		if true == flags.stdio {
			log.Printf("remote: using stdio")
			upstream, err = ConnectStdio(os.Stdin, os.Stdout)

			if nil != err {
				log.Fatalf("error[stdio] %v\n", err)
			}
		} else if true == flags.tls {
			// setup remote TLS communication
			log.Printf("remote: using TCP w/TLS")
			upstream, err = ConnectTLS(flags.remote)

			if nil != err {
				log.Printf("error[tcp/tls] %v\n", err)
				return nil, err
			}
		} else {
			// setup remote TCP communication without TLS
			log.Printf("remote: using TCP (NO-TLS)")
			upstream, err = ConnectTCP(flags.remote)

			if nil != err {
				log.Printf("error[tcp] %v\n", err)
				return nil, err
			}
		}

		if nil == upstream {
			return nil, errors.New("no upstream configured")
		}

		return upstream, nil
	}

	upstream, err = connectUpstream()

	downstreamchan := downstream.Open()
	upstreamchan := upstream.Open()

	var maxRetries = 0

	if true == flags.reconnect {
		maxRetries = -1
	}

	var upstreamBackoff Backoff = NewBackoff(time.Second, 2.0, maxRetries)
	var downstreamBackoff Backoff = NewBackoff(time.Second, 2.0, maxRetries)

	for {
		select {
		case buf, state := <-downstreamchan:
			if false == state {
				if true == flags.reconnect {
					log.Printf("downstream:close, reconnecting\n")
					downstreamBackoff.Until(func() error {
						downstream, err = ConnectSerial(path)
						if nil != err {
							log.Printf("error[downstream:connect] %v\n", err)
						} else {
							log.Printf("downstream:connect reconnected\n")
						}
						return err
					})

					downstreamchan = downstream.Open()
				} else {
					log.Printf("downstream:close, terminating\n")
					return
				}
			} else {
				log.Printf("downstream:recv[%v] %v\n", state, buf)
				upstream.Write(buf, -1)
			}

		case buf, state := <-upstreamchan:
			if false == state {
				if true == flags.reconnect {
					log.Printf("upstream:close, reconnecting\n")
					upstreamBackoff.Until(func() error {
						upstream, err = connectUpstream()
						if nil != err {
							log.Printf("error[upstream:connect] %v\n", err)
						} else {
							log.Printf("upstream:connect reconnect\n")
						}
						return err
					})

					upstreamchan = upstream.Open()
				} else {
					log.Printf("upstream:close, terminating\n")
					return
				}

			} else {
				log.Printf("upstream:recv[%v] %v\n", state, buf)
				if true == flags.stdio && buf[0] == 10 {
					downstream.Write([]byte("\x0a\x00\x00\x00\x00\x03\x03\x10\x00\x00"), -1)
				}

				downstream.Write(buf, -1)
			}
		}
	}
}
